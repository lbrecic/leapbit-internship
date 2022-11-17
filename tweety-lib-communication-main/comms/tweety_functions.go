package comms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	tw "gitlab.com/leapbit-practice/tweety-lib-twitter/twitter"
)

const (
	httpCounterEndpoint         = "user_ids"
	httpDBSaverMetadataEndpoint = "user_metadata"
	httpDBSaverExistsEndpoint   = "user_exists"
	httpDBSaverLocationEndpoint = "location"
)

// Function for communication between Tweety-Collector and Tweety-Counter.
// Specifically, function sends data from Collector to Counter via HTTP request.
func SendIdsDataToCounter(ids []string, c *http.Client, addr string, port string) (*http.Response, error) {
	url := fmt.Sprintf("%s:%s/%s", addr, port, httpCounterEndpoint)
	friendsReq := tw.ReqFriends{
		Friends_ids: ids,
	}
	resp, err := request("SendIdsDataToCounter", c, http.MethodPost, url, friendsReq)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// Function for communication between Tweety-Collector and Tweety-DBSaver.
// Specifically, function sends data from Collector to DBSaver via HTTP request.
func SendUserDataToDatabase(user ReqUser, c *http.Client, addr string, port string) (*http.Response, error) {
	url := fmt.Sprintf("%s:%s/%s", addr, port, httpDBSaverMetadataEndpoint)
	resp, err := request("SendUserDataToDatabase", c, http.MethodPost, url, user)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// Function for communication between Tweety-Collector and Tweety-DBSaver.
// Specifically, Collector checks with DBSaver if user already exists in database via HTTP request.
func CheckIfExists(userId ReqUserId, c *http.Client, addr string, port string) (*http.Response, RespUserExists, error) {
	var exists RespUserExists
	url := fmt.Sprintf("%s:%s/%s", addr, port, httpDBSaverExistsEndpoint)
	resp, err := request("CheckIfExists", c, http.MethodGet, url, userId)
	if err != nil {
		return resp, exists, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, exists, fmt.Errorf("%sCheckIfExists method body reading error: \n%s%s", space, space, err)
	}
	err = json.Unmarshal(body, &exists)
	if err != nil {
		return resp, exists, fmt.Errorf("%sCheckIfExists method unmarshalling error: \n%s%s", space, space, err)
	}
	return resp, exists, nil
}

func SendLocationDataToDatabase(locInfo ReqLocationForDB, c *http.Client, addr string, port string) (*http.Response, error) {
	url := fmt.Sprintf("%s:%s/%s", addr, port, httpDBSaverLocationEndpoint)
	resp, err := request("SendLocationDataToDatabase", c, http.MethodPost, url, locInfo)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func request(funcName string, c *http.Client, method string, requestURL string, data interface{}) (*http.Response, error) {
	reqData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("%s%s method marshalling error: \n%s%s", space, funcName, space, err)
	}
	req, err := http.NewRequest(method, requestURL, bytes.NewBufferString(string(reqData)))
	if err != nil {
		return nil, fmt.Errorf("%s%s method new request error: \n%s%s", space, funcName, space, err)
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return resp, fmt.Errorf("%s%s method server communication error: \n%s%s", space, funcName, space, err)
	}
	c.CloseIdleConnections()
	return resp, nil
}
