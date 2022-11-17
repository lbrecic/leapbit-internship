// Package main initializes and run Tweety-Collector
// application and its methods.
package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	com "gitlab.com/leapbit-practice/tweety-lib-communication/comms"
	tw "gitlab.com/leapbit-practice/tweety-lib-twitter/twitter"
)

const (
	version = "1.10.0"
	author  = "Luka Brecic"
	appname = "Tweety-Collector"
)

// Tweety-Collector application structure.
// Contains clients for communications with Twitter
// and other microservices.
type App struct {
	Logger           *com.TweetyLogger
	TwitterClient    *HTTPClientTwitter
	CounterClient    *HTTPClientCounter
	DBSaverClient    *HTTPClientDBSaver
	WorkersWaitGroup sync.WaitGroup
	Metric
	Session
}

// Structure Session application internal queue,
// how many friends to search and how many friends to download.
type Session struct {
	Queue    []string
	Treshold int
	Friends  int
	Workers  int
}

// Function initializes Tweety-Collector application and clients
// based on loaded configuration parameters.
func appInit(config *Config, timeStart string) *App {
	timeStart = strings.ReplaceAll(timeStart, " ", "_")
	timeStart = strings.ReplaceAll(timeStart, ":", "_")
	fileName := fmt.Sprintf("%s%s", "log_", timeStart)
	logger := com.NewTweetyLogger(fileName, config.LogDir, config.LogLevel)
	app := NewCollectorApp(config, logger)
	app.initializeWorkers()
	app.shutdownAwait()
	return app
}

// Tweety-Collector application constructor.
func NewCollectorApp(config *Config, logger *com.TweetyLogger) *App {
	app := &App{
		Logger:           logger,
		TwitterClient:    NewTwitterClient(config.Bearer),
		CounterClient:    NewCounterClient(config.CounterAddr, config.CounterPort, logger),
		DBSaverClient:    NewDBSaverClient(config.DBSaverAddr, config.DBSaverPort, logger),
		WorkersWaitGroup: sync.WaitGroup{},
		Metric:           NewMetric(),
		Session: Session{
			Queue:    make([]string, 0),
			Treshold: config.Treshold,
			Friends:  config.Friends,
			Workers:  config.Workers,
		},
	}
	return app
}

// Method marks starting point of Tweety-Collector application.
// Once run, method can only be interrupted by internal error or SIGINT.
func (app *App) start(username string) {
	app.Logger.LogData(com.CLEAN, "%s%s", delimiter, delimiter)
	app.Logger.LogData(com.INFO, "Application started. Data scraping will begin shortly.")
	app.Logger.LogData(com.CLEAN, "%s%s", delimiter, delimiter)
	go app.CounterClient.clearCache()
	for {
		id, err := app.process(username, true)
		if err != nil {
			if err.Error() == "401" {
				time.Sleep(1 * time.Hour)
			} else if err.Error() == "429" {
				time.Sleep(5 * time.Minute)
			} else if err.Error() != "" {
				app.Logger.LogData(com.ERROR, "start method error: %s", err.Error())
			}
			continue
		}
		if len(app.Queue) == 0 {
			return
		}
		app.CounterClient.DataChannel <- []string{id}
		app.session()
	}
}

// Method processes users and sends their data to Tweety-Counter and Tweety-DBSaver applications.
func (app *App) session() {
	for {
		timer := prometheus.NewTimer(app.MethodDurations.WithLabelValues("session"))
		ids := make([]string, 0)
		for i := 0; i < app.Friends; {
			userId := app.Queue[0]
			_, err := app.process(userId, false)
			if err != nil {
				if err.Error() == "401" {
					time.Sleep(1 * time.Hour)
				} else if err.Error() == "429" {
					time.Sleep(5 * time.Minute)
				} else if err.Error() != "" {
					app.Logger.LogData(com.ERROR, err.Error())
					time.Sleep(500 * time.Millisecond)
				}
				continue
			}
			app.Queue = app.Queue[1:]
			ids = append(ids, userId)
			i++
		}
		if len(ids) > 0 {
			app.CounterClient.DataChannel <- ids
		}
		timer.ObserveDuration()
		time.Sleep(15 * time.Minute)
	}
}

// Method scrapes metadata and friends ids for given usrename/user id.
func (app *App) process(userId string, init bool) (string, error) {
	timer := prometheus.NewTimer(app.MethodDurations.WithLabelValues("process"))
	var query string
	// Check if it is initial process method call
	if init {
		query = fmt.Sprintf("screen_name=%s", userId)
	} else {
		resp, userExists, err := app.DBSaverClient.userExists(userId)
		app.HttpRequests.WithLabelValues("dbsaver", "user_exists").Inc()
		if err != nil {
			timer.ObserveDuration()
			return "", fmt.Errorf("databaseUserSender function error: %s", err)
		} else if resp.StatusCode != http.StatusOK {
			timer.ObserveDuration()
			return "", fmt.Errorf("databaseUserSender function response error: %s", http.StatusText(resp.StatusCode))
		}
		if userExists.Exists && time.Since(userExists.Last_modified) <= 1*time.Hour {
			app.Queue = app.Queue[1:]
			timer.ObserveDuration()
			return "", fmt.Errorf("")
		}
		query = fmt.Sprintf("user_id=%s", userId)
	}
	// Twitter API metadata scraping for user
	user, resp, err := tw.UserGetMetadata(query, &app.TwitterClient.Client, app.TwitterClient.Bearer)
	app.HttpRequests.WithLabelValues("twitter", "user_metadata").Inc()
	if err != nil {
		timer.ObserveDuration()
		return "", fmt.Errorf("twitter API error: %s", err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		app.Logger.LogData(com.INFO, "Twitter API interruption. Server response: %s", http.StatusText(resp.StatusCode))
		timer.ObserveDuration()
		return "", fmt.Errorf("401")
	}
	// Twitter API getting friends ids for user
	friendsIds, resp, err := tw.UserGetFriends(user[0].Screen_name, &app.TwitterClient.Client, app.TwitterClient.Bearer)
	app.HttpRequests.WithLabelValues("twitter", "friends_ids").Inc()
	if err != nil {
		timer.ObserveDuration()
		return user[0].Id_str, fmt.Errorf("twitter API error: %s", err.Error())
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		app.Logger.LogData(com.INFO, "Waiting on Twitter server response! Please, be patient, this could exceed up to max 15 minutes. status error: %s", http.StatusText(resp.StatusCode))
		timer.ObserveDuration()
		return user[0].Id_str, fmt.Errorf("429")
	}
	if resp.StatusCode != http.StatusOK {
		app.Logger.LogData(com.INFO, "Twitter API interruption. Server response: %s", http.StatusText(resp.StatusCode))
		timer.ObserveDuration()
		return "", fmt.Errorf("401")
	}
	// Slicing to wanted number of friends ids
	if len(friendsIds) >= int(app.Friends) {
		friendsIds = friendsIds[:app.Friends]
	}
	// Checking if it is allowed to put this session friends into queue for further processing
	if len(app.Queue)+len(friendsIds) <= app.Treshold {
		app.Queue = append(app.Queue, friendsIds...)
	}
	app.Logger.LogData(com.INFO, "%s's processed!Number of friends downloaded: %d", user[0].Screen_name, len(friendsIds))
	app.DBSaverClient.UserChannel <- createUser(user[0], friendsIds)
	if user[0].Location != "" {
		app.DBSaverClient.LocationChannel <- createUserLocationPair(user[0].Id_str, user[0].Location)
	}
	timer.ObserveDuration()
	return user[0].Id_str, nil
}
