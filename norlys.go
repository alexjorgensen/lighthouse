package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"
)

// NorlysAPI contains all functions needed to get pricing information from Norlys
type NorlysAPI struct {
}

// NorlysPricingResult contains the prices in DKK øre for the Date specified in PriceDate
// Sector DK1 is West denmark and DK2 is east denmark
type NorlysPricingResult struct {
	PriceDate     time.Time `json:"PriceDate"`
	Sector        string    `json:"Sector"`
	Currency      string    `json:"Currency"`
	DisplayPrices []struct {
		Time  string  `json:"Time"`
		Value float64 `json:"Value"`
	} `json:"DisplayPrices"`
}

// GetPrices Makes a HTTP request towards the norlys API, and returns the FlexEl prices.
func (n *NorlysAPI) GetPrices(numberOfDays int, settings *Settings) (res []NorlysPricingResult, err error) {
	res = make([]NorlysPricingResult, 0)

	// Generate the URL
	url := settings.NorlysAPI.URL + "days=" + strconv.Itoa(numberOfDays) + "&sector=DK1"

	// Make the HTTP call towards the Norlys API
	resp, err := http.Get(url)
	if err != nil {
		return res, err
	}

	// check response code
	if resp.StatusCode >= 300 {
		return res, errors.New("norlys API returned status:" + resp.Status)
	}

	// Check the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return res, err
	}

	// parse the json response from norlys API
	err = json.Unmarshal(body, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}
