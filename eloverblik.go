package main

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// ElOverblik struct handles all communication towards ElOverblik datahub
// it's able to fetch the metering timeSeries data
type ElOverblik struct {
	ApplicationToken struct {
		Token  string
		Expire time.Time
	}
	RequestToken struct {
		Token  string
		Expire time.Time
	}
}

// GetRequestToken is using the application token to get a request token
// is a request is successfull, it updates the RequestToken struct in the
// ElOverblik main struct
// If the current Request token hasent expired then it wont fetch a new one,
// unless forceGetToken is set to true
// There is a limitation on how many requests, you are allowed to call the eloverblik /api/token
func (eo *ElOverblik) GetRequestToken(forceGetToken bool) error {

	// check if Application token is configured, and not expired
	if eo.ApplicationToken.Token == "" {
		return errors.New("no Application token configured")
	}
	if eo.ApplicationToken.Expire.Before(time.Now()) {
		return errors.New("application token has expired, please update it on eloverblik.dk website")
	}

	// create the context, timeout after 20 seconds
	timeoutContext, cancelFunc := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelFunc()

	// create the request
	req, err := http.NewRequestWithContext(timeoutContext, http.MethodGet, "https://api.eloverblik.dk/customerapi/api/token", nil)
	if err != nil {
		return err
	}

	// add the headers needed
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+eo.ApplicationToken.Token)

	// make the http request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode > 299 {
		return errors.New("unable to get request token, server responded:" + res.Status)
	}

	return nil
}
