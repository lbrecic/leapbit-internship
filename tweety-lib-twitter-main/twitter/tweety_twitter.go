// Package provides functions for communication between
// Twitter API 1.1 and Tweety aplication micro-services.
package twitter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const (
	httpLookupRequestTemplate     = "https://api.twitter.com/1.1/users/lookup.json?%s"
	httpFriendsRequestTemplate    = "https://api.twitter.com/1.1/friends/ids.json?stringify_ids=true&screen_name=%s"
	httpImageUrlsRequestTemplate  = "https://api.twitter.com/1.1/users/show.json?user_id=%s"
	httpStatusesRequestTemplate   = "https://api.twitter.com/1.1/statuses/user_timeline.json?user_id=%v&count=%v&tweet_mode=extended"
	httpStatusInfoRequestTemplate = "https://twitter.com/%v/status/%v"
)

type RespTwitterApiUser struct {
	Id          uint64 `json:"id"`
	Id_str      string `json:"id_str"`
	Name        string `json:"name"`
	Screen_name string `json:"screen_name"`
	Location    string `json:"location"`
	URL         string `json:"url"`
	Entities    struct {
		Urls []struct {
			Url string `json:"expanded_url"`
		} `json:"urls"`
	} `json:"entities"`
	Description     string      `json:"description"`
	Protected       bool        `json:"protected"`
	Verified        bool        `json:"verified"`
	Followers_count uint64      `json:"followers_count"`
	Friends_count   uint64      `json:"friends_count"`
	Statuses_count  uint64      `json:"statuses_count"`
	Created_at      TwitterTime `json:"created_at"`
}

type RespTwitterApiTweet struct {
	Created_at TwitterTime `json:"created_at"`
	Id         uint64      `json:"id"`
	Id_str     string      `json:"id_str"`
	Text       string      `json:"full_text"`
	Url        string
	User       struct {
		Id          uint64 `json:"id"`
		Screen_name string `json:"screen_name"`
	} `json:"user"`
}

type RespTwitterApiFriends struct {
	Friends_ids []string `json:"ids"`
}

type RespTwitterApiImages struct {
	UrlProfileImage string `json:"profile_image_url_https"`
	UrlBanner       string `json:"profile_banner_url"`
}

type ReqFriends struct {
	Friends_ids []string `json:"ids"`
}

type RespFriends struct {
	Friends_ids []string `json:"ids"`
}

type RespDoneFriends struct {
	Friends_ids []string `json:"ids"`
}

type TwitterTime struct {
	time.Time
}

func (twTime *TwitterTime) UnmarshalJSON(b []byte) error {
	bString := string(b)
	bString, err := strconv.Unquote(bString)
	if err != nil {
		return err
	}
	myTime, err := time.Parse(time.RubyDate, bString)
	if err != nil {
		return err
	}
	*twTime = TwitterTime{myTime}

	return nil
}

func (twTime TwitterTime) MarshalJSON() ([]byte, error) {
	return []byte("\"" + twTime.Format(time.RubyDate) + "\""), nil
}

func UserGetImageUrls(userId string, c *http.Client, bearer string) (RespTwitterApiImages, error, error) {
	var respImages RespTwitterApiImages

	req, errMsg := http.NewRequest("GET", fmt.Sprintf(httpImageUrlsRequestTemplate, userId), nil)
	if errMsg != nil {
		return respImages, nil, fmt.Errorf("cannot create request. Error: %s", errMsg.Error())
	}
	req.Header.Add("Authorization", bearer)
	resp, err := c.Do(req)
	if err != nil {
		return respImages, fmt.Errorf("cannot do the given request. Error: %s", err.Error()), nil
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	errMsg = json.Unmarshal(body, &respImages)
	if errMsg != nil {
		return respImages, nil, fmt.Errorf("cannot unmarshal body. Error: %s", errMsg.Error())
	}

	return respImages, nil, nil

}

// Function retrieves up to tweet_no tweets for given users.
// Users are reached by their ids. Notice that authorization token is required.
func UserGetTweets(userId string, tweetNo uint64, c *http.Client, bearer string) ([]RespTwitterApiTweet, error, error) {
	var userTweets []RespTwitterApiTweet

	req, errMsg := http.NewRequest("GET", fmt.Sprintf(httpStatusesRequestTemplate, userId, tweetNo), nil)
	if errMsg != nil {
		return nil, nil, fmt.Errorf("cannot create request. Error: %s", errMsg.Error())
	}
	req.Header.Add("Authorization", bearer)
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot do the given request. Error: %s", err.Error()), nil
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	errMsg = json.Unmarshal(body, &userTweets)
	if errMsg != nil {
		return nil, nil, fmt.Errorf("cannot unmarshal body. Error: %s", errMsg.Error())
	}
	for i := 0; i < len(userTweets); i++ {
		userTweets[i].Url = fmt.Sprintf(httpStatusInfoRequestTemplate, userTweets[i].User.Screen_name, userTweets[i].Id_str)
	}
	return userTweets, nil, nil
}

// Function retrieves metadata specified by Twitter API 1.1
// for given user. Notice that authorization token is required.
func UserGetMetadata(query string, c *http.Client, bearer string) ([]RespTwitterApiUser, *http.Response, error) {
	var u []RespTwitterApiUser
	http_lookupRequest := fmt.Sprintf(httpLookupRequestTemplate, query)
	req, err := http.NewRequest(http.MethodGet, http_lookupRequest, nil)
	if err != nil {
		return u, nil, fmt.Errorf("UserGetMetadata method new request error: %s", err)
	}
	req.Header.Add("Authorization", bearer)
	resp, err := c.Do(req)
	if err != nil {
		return u, resp, fmt.Errorf("UserGetMetadata method server communication error: %s", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return u, resp, fmt.Errorf("UserGetMetadata method body reading error: %s", err)
	}
	err = json.Unmarshal(body, &u)
	if err != nil {
		return u, resp, fmt.Errorf("UserGetMetadata method unmarshalling error: %s", err)
	}
	return u, resp, nil
}

// Function retrieves array of friends ids for given user.
// User is referenced by its screen name. Notice that authorization token is required.
func UserGetFriends(screen_name string, client *http.Client, bearer string) ([]string, *http.Response, error) {
	var friendsResp RespTwitterApiFriends
	http_friendsRequest := fmt.Sprintf(httpFriendsRequestTemplate, screen_name)
	req, err := http.NewRequest(http.MethodGet, http_friendsRequest, nil)
	if err != nil {
		return friendsResp.Friends_ids, nil, fmt.Errorf("UserGetFriends method new request error: %s", err)
	}
	req.Header.Add("Authorization", bearer)
	resp, err := client.Do(req)
	if err != nil {
		return friendsResp.Friends_ids, resp, fmt.Errorf("UserGetFriends method server communication error: %s", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return friendsResp.Friends_ids, resp, fmt.Errorf("UserGetFriends method body reading error: %s", err)
	}
	err = json.Unmarshal(body, &friendsResp)
	if err != nil {
		return friendsResp.Friends_ids, resp, fmt.Errorf("UserGetFriends method unmarshalling error: %s", err)
	}
	return friendsResp.Friends_ids, resp, nil
}
