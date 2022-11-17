// Package main initializes and run Tweety-Collector
// application and its methods.
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	com "gitlab.com/leapbit-practice/tweety-lib-communication/comms"
)

const (
	httpRESTCountriesAPIBaseURL            = "https://restcountries.eu/rest/v2/%s"
	httpCountryNameRequestTemplate         = "name/%s"
	httpCountryFullNameRequestTemplate     = "name/%s?fullText=true"
	httpCountryCodeRequestTemplate         = "alpha/%s"
	httpCountryRegionRequestTemplate       = "region/%s"
	httpCountryCapitalRequestTemplate      = "capital/%s"
	httpCountryRegionalBlocRequestTemplate = "regionalbloc/%s"
)

var (
	endpoints []string
)

// Function init starts at build; Creates array of endpoints for REST Countries API v2.
func init() {
	endpoints = []string{httpCountryNameRequestTemplate, httpCountryFullNameRequestTemplate,
		httpCountryCodeRequestTemplate, httpCountryRegionRequestTemplate,
		httpCountryCapitalRequestTemplate, httpCountryRegionalBlocRequestTemplate}
}

// Method tries to collect location data for given input at any of endpoints for REST Countries API v2.
func (client *HTTPClientDBSaver) location(loc string) (com.RespLocation, *http.Response, error) {
	for no, endpoint := range endpoints {
		locationResp, resp, err := client.requestSending(endpoint, loc)
		if err != nil {
			return locationResp, resp, fmt.Errorf("location method %d. error: %s", no+1, err)
		}
		if resp.StatusCode == http.StatusOK {
			return locationResp, resp, nil
		}
	}
	return com.RespLocation{}, &http.Response{StatusCode: http.StatusNotFound}, nil
}

// Method forms location data request for given endpoint and location name
// and sends it to getLocationInfo method.
func (client *HTTPClientDBSaver) requestSending(endpoint string, loc string) (com.RespLocation, *http.Response, error) {
	var locationResp []com.RespLocation
	requestEndpoint := fmt.Sprintf(endpoint, loc)
	requestURL := fmt.Sprintf(httpRESTCountriesAPIBaseURL, requestEndpoint)
	locationResp, resp, err := client.getLocationInfo(requestURL)
	if len(locationResp) > 0 {
		return locationResp[0], resp, err
	}
	return com.RespLocation{}, resp, err

}

// Method for request processing and scraping location data from REST Countries API v2.
func (client *HTTPClientDBSaver) getLocationInfo(requestURL string) ([]com.RespLocation, *http.Response, error) {
	var locationResp []com.RespLocation
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return locationResp, nil, fmt.Errorf("%sgetLocationInfo method new request error: %s", space, err)
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Client.Do(req)
	if err != nil {
		return locationResp, resp, fmt.Errorf("%sgetLocationInfo method server communication error", space)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return locationResp, resp, fmt.Errorf("%sgetLocationInfo method body reading error: %s", space, err)
		}
		err = json.Unmarshal(body, &locationResp)
		if err != nil {
			return locationResp, resp, fmt.Errorf("%sgetLocationInfo method unmarshalling error: %s", space, err)
		}
	}
	return locationResp, resp, nil
}

// Method parses locations that are separated
// by comma or whitespace.
func splitLocation(location string) []string {
	locations := []string{location}
	if strings.Contains(location, ", ") {
		locations = append(locations, strings.Split(location, ", ")...)
	} else if strings.Contains(location, " ") {
		locations = append(locations, strings.Split(location, " ")...)
	}
	return locations
}
