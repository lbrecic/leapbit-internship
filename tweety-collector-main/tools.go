// Package main initializes and run Tweety-Collector
// application and its methods.
package main

import (
	"time"

	com "gitlab.com/leapbit-practice/tweety-lib-communication/comms"
	tw "gitlab.com/leapbit-practice/tweety-lib-twitter/twitter"
)

const (
	delimiter = "=========================================================================="
	space     = "\n                             "
)

// Structure UserLocationPair holds
// location name for given user id.
type UserLocationPair struct {
	UserId       string
	LocationName string
}

// Function for creating User struct variable.
func createUser(apiUser tw.RespTwitterApiUser, ids []string) com.ReqUser {
	user := com.ReqUser{
		Id:              apiUser.Id,
		Id_str:          apiUser.Id_str,
		Name:            apiUser.Name,
		Screen_name:     apiUser.Screen_name,
		Location:        apiUser.Location,
		URL:             apiUser.URL,
		Description:     apiUser.Description,
		Protected:       apiUser.Protected,
		Verified:        apiUser.Verified,
		Followers_count: apiUser.Followers_count,
		Friends_count:   apiUser.Friends_count,
		Statuses_count:  apiUser.Statuses_count,
		Created_at:      apiUser.Created_at.Time,
		Followers_id:    ids,
		App_name:        appname,
		Sent_at:         time.Now(),
	}
	return user
}

// Function for creating UserLocationPair struct variable.
func createUserLocationPair(userId string, locationName string) UserLocationPair {
	pair := UserLocationPair{
		UserId:       userId,
		LocationName: locationName,
	}
	return pair
}
