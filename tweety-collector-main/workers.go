// Package main initializes and run Tweety-Collector
// application and its methods.
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	com "gitlab.com/leapbit-practice/tweety-lib-communication/comms"
	tw "gitlab.com/leapbit-practice/tweety-lib-twitter/twitter"
)

// Method initializes workers as separate goroutines.
func (app *App) initializeWorkers() {
	for i := 0; i < int(app.Workers); i++ {
		go app.CounterClient.counterIdsSenderWorker(app)
		go app.DBSaverClient.databaseUserSenderWorker(app)
		go app.DBSaverClient.databaseLocationSenderWorker(app)
	}
}

// Tweety-Counter client worker method listens
// for user ids on clients built-in channel.
func (client *HTTPClientCounter) counterIdsSenderWorker(app *App) {
	app.WorkersWaitGroup.Add(1)
	for ids := range client.DataChannel {
		timer := prometheus.NewTimer(app.MethodDurations.WithLabelValues("counterIdsSenderWorker"))
		for {
			failedIds, err := client.counterIdsSender(ids)
			app.HttpRequests.WithLabelValues("counter", "friends_ids").Inc()
			if err != nil {
				client.Logger.LogData(com.ERROR, err.Error())
				continue
			}
			if len(failedIds) > 0 {
				resend := make([]string, 0)
				client.CacheLock.Lock()
				for _, id := range failedIds {
					e, ok := client.Cache.Ids[id]
					if !ok {
						client.Cache.Ids[id] = 0
					} else {
						if e < 3 {
							resend = append(resend, id)
							client.Cache.Ids[id] += 1
						} else {
							delete(client.Cache.Ids, id)
						}
					}
				}
				client.CacheLock.Unlock()
				ids = resend
				if len(ids) > 0 {
					continue
				}
			}
			break
		}
		timer.ObserveDuration()
	}
	app.WorkersWaitGroup.Done()
}

// Tweety-DBSaver client worker method listens
// for users on clients built-in channel.
func (client *HTTPClientDBSaver) databaseUserSenderWorker(app *App) {
	app.WorkersWaitGroup.Add(1)
	for user := range client.UserChannel {
		timer := prometheus.NewTimer(app.MethodDurations.WithLabelValues("databaseUserSenderWorker"))
		for {
			err := client.databaseUserSender(user)
			app.HttpRequests.WithLabelValues("dbsaver", "user_metadata").Inc()

			if err != nil {
				client.Logger.LogData(com.ERROR, err.Error())
				continue
			}
			break
		}
		timer.ObserveDuration()
	}
	app.WorkersWaitGroup.Done()
}

// Tweety-DBSaver client worker method listens
// for locations on clients built-in channel.
func (client *HTTPClientDBSaver) databaseLocationSenderWorker(app *App) {
	app.WorkersWaitGroup.Add(1)
	for pair := range client.LocationChannel {
		timer := prometheus.NewTimer(app.MethodDurations.WithLabelValues("databaseLocationSenderWorker"))
		for {
			locations := splitLocation(pair.LocationName)
			var locErr error
			for _, loc := range locations {
				respLoc, resp, err := client.location(loc)
				app.HttpRequests.WithLabelValues("countries", "location").Inc()
				if err != nil {
					client.Logger.LogData(com.ERROR, err.Error())
					continue
				}
				if resp.StatusCode != http.StatusOK {
					client.Logger.LogData(com.INFO, "Location not found for %s connected to id %s. status code: %d", loc, pair.UserId, resp.StatusCode)
					continue
				}
				client.Logger.LogData(com.INFO, "Location data for %s obtained!", pair.LocationName)
				locErr = client.databaseLocationSender(respLoc, pair.UserId)
				app.HttpRequests.WithLabelValues("dbsaver", "location").Inc()
				break
			}
			if locErr != nil {
				client.Logger.LogData(com.ERROR, locErr.Error())
				continue
			}
			break
		}
		timer.ObserveDuration()
	}
	app.WorkersWaitGroup.Done()
}

// Method handles user ids data sending to Tweety-Counter server.
func (client *HTTPClientCounter) counterIdsSender(ids []string) ([]string, error) {
	var counterRespIds tw.RespFriends
	resp, err := com.SendIdsDataToCounter(ids, &client.Client, client.Addr, client.Port)
	if err != nil {
		return counterRespIds.Friends_ids, fmt.Errorf("counterIdsSender function error: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return counterRespIds.Friends_ids, fmt.Errorf("counterIdsSender function error: %s", http.StatusText(resp.StatusCode))
	}
	client.Logger.LogData(com.INFO, "Session friends ids data successfully sent to Tweety-Counter server!")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return counterRespIds.Friends_ids, fmt.Errorf("counterIdsSender function error: %s", err)
	}
	err = json.Unmarshal(body, &counterRespIds)
	if err != nil {
		return counterRespIds.Friends_ids, fmt.Errorf("counterIdsSender function error: %s", err)
	}
	return counterRespIds.Friends_ids, nil
}

// Method handles user data sending to Tweety-DBSaver server.
func (client *HTTPClientDBSaver) databaseUserSender(user com.ReqUser) error {
	resp, err := com.SendUserDataToDatabase(user, &client.Client, client.Addr, client.Port)
	if err != nil {
		return fmt.Errorf("databaseUserSender function error: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("databaseUserSender function error: %s", http.StatusText(resp.StatusCode))
	}
	client.Logger.LogData(com.INFO, "%s's data successfully sent to Tweety-DBSaver server!", user.Name)
	return nil
}

// Method handles location data sending to Tweety-DBSaver server.
func (client *HTTPClientDBSaver) databaseLocationSender(loc com.RespLocation, userId string) error {
	locationInfo := com.ReqLocationForDB{
		LocationInfo: loc,
		UserId:       userId,
		AppName:      appname,
		SentAt:       time.Now(),
	}
	resp, err := com.SendLocationDataToDatabase(locationInfo, &client.Client, client.Addr, client.Port)
	if err != nil {
		return fmt.Errorf("databaseLocationSender function error: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("databaseLocationSender function error: %s", http.StatusText(resp.StatusCode))
	}
	client.Logger.LogData(com.INFO, "%s's location data successfully sent to Tweety-DBSaver server!", loc.Name)
	return nil
}

// Method for handling response from database while checking
// if user exists.
func (client *HTTPClientDBSaver) userExists(id string) (*http.Response, com.RespUserExists, error) {
	userId := com.ReqUserId{
		UserId:  id,
		AppName: appname,
		SentAt:  time.Now(),
	}
	resp, userExists, err := com.CheckIfExists(userId, &client.Client, client.Addr, client.Port)
	return resp, userExists, err
}
