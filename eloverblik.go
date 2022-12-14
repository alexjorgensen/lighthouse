package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/brianvoe/sjwt"
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

type ElOverblikTokenResult struct {
	Result string `json:"result"`
}

// SetApplicationToken Checks if token has expired, if not it sets the application token
func (eo *ElOverblik) SetApplicationToken(token string) error {
	claims, _ := sjwt.Parse(token)

	// check if the token has expired.
	exp, err := claims.GetStr("exp")
	if err != nil {
		return err
	}

	// convert the exp string to integer
	expire, err := strconv.ParseInt(exp, 10, 64)
	if err != nil {
		return errors.New("unable to convert token expire string to int64: " + err.Error())
	}
	// check if the token has expired
	if time.Unix(expire, 0).Before(time.Now()) {
		return errors.New("application token has expired")
	}

	eo.ApplicationToken.Token = token
	eo.ApplicationToken.Expire = time.Unix(expire, 0)
	return nil
}

// GetRequestToken is using the application token to get a request token
// is a request is successfully, it updates the RequestToken struct in the
// ElOverblik main struct
// If the current Request token hasn't expired then it won't fetch a new one,
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

	// check if the current request token has expired
	if eo.RequestToken.Token != "" && eo.RequestToken.Expire.After(time.Now()) && !forceGetToken {
		return errors.New("the current token is still valid")
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

	// get the body from the http response
	var tokenRes ElOverblikTokenResult
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// Unmarshal the json into the result struct
	err = json.Unmarshal(body, &tokenRes)
	if err != nil {
		return err
	}

	// we received a token from the server, let's check the content of the token
	claims, _ := sjwt.Parse(tokenRes.Result)

	// Get claims
	exp, err := claims.GetStr("exp") // John Doe
	if err != nil {
		return err
	}

	// convert the exp string to int64
	expire, err := strconv.ParseInt(exp, 10, 64)
	if err != nil {
		return errors.New("unable to convert token expire string to int64: " + err.Error())
	}
	// check if the token has expired
	if time.Unix(expire, 0).Before(time.Now()) {
		return errors.New("request token has expired")
	}

	eo.RequestToken.Token = tokenRes.Result
	eo.RequestToken.Expire = time.Unix(expire, 0)

	return nil
}
