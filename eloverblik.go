package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/brianvoe/sjwt"
)

// ElOverblik struct handles all communication towards ElOverblik datahub
// it's able to fetch the metering timeSeries data
type ElOverblik struct {
	Lock             sync.RWMutex
	ApplicationToken struct {
		Token  string
		Expire time.Time
	}
	RequestToken struct {
		Token  string
		Expire time.Time
	}
	MeteringPoints []EloverblikMeteringPoint
}

// EloverblikMeteringPointResult this is the result returned when calling the
// /api/meteringpoints/meteringpoints API call
type EloverblikMeteringPointResult struct {
	Result []EloverblikMeteringPoint `json:"result"`
}

// EloverblikMeteringPoint holds information about meteringpoints
type EloverblikMeteringPoint struct {
	StreetCode              string        `json:"streetCode"`
	StreetName              string        `json:"streetName"`
	BuildingNumber          string        `json:"buildingNumber"`
	FloorId                 int           `json:"floorId"`
	RoomId                  int           `json:"roomId"`
	CitySubDivisionName     string        `json:"citySubDivisionName"`
	MunicipalityCode        string        `json:"municipalityCode"`
	LocationDescription     string        `json:"locationDescription"`
	SettlementMethod        string        `json:"settlementMethod"`
	MeterReadingOccurrence  string        `json:"meterReadingOccurrence"`
	FirstConsumerPartyName  string        `json:"firstConsumerPartyName"`
	SecondConsumerPartyName string        `json:"secondConsumerPartyName"`
	MeterNumber             string        `json:"meterNumber"`
	ConsumerStartDate       time.Time     `json:"consumerStartDate"`
	MeteringPointId         string        `json:"meteringPointId"`
	TypeOfMP                string        `json:"typeOfMP"`
	BalanceSupplierName     string        `json:"balanceSupplierName"`
	Postcode                string        `json:"postcode"`
	CityName                string        `json:"cityName"`
	HasRelation             bool          `json:"hasRelation"`
	ConsumerCVR             string        `json:"consumerCVR"`
	DataAccessCVR           string        `json:"dataAccessCVR"`
	ChildMeteringPoints     []interface{} `json:"childMeteringPoints"`
}

type EloverblikMeteringTimeSeriesResult struct {
	Result []struct {
		MyEnergyDataMarketDocument struct {
			MRID               string    `json:"mRID"`
			CreatedDateTime    time.Time `json:"createdDateTime"`
			PeriodTimeInterval struct {
				Start time.Time `json:"start"`
				End   time.Time `json:"end"`
			} `json:"period.timeInterval"`
			TimeSeries []struct {
				MRID                  string `json:"mRID"`
				BusinessType          string `json:"businessType"`
				CurveType             string `json:"curveType"`
				MeasurementUnitName   string `json:"measurement_Unit.name"`
				MarketEvaluationPoint struct {
					MRID struct {
						CodingScheme string `json:"codingScheme"`
						Name         string `json:"name"`
					} `json:"mRID"`
				} `json:"MarketEvaluationPoint"`
				Period []struct {
					Resolution   string `json:"resolution"`
					TimeInterval struct {
						Start time.Time `json:"start"`
						End   time.Time `json:"end"`
					} `json:"timeInterval"`
					Point []struct {
						Position            string `json:"position"`
						OutQuantityQuantity string `json:"out_Quantity.quantity"`
						OutQuantityQuality  string `json:"out_Quantity.quality"`
					} `json:"Point"`
				} `json:"Period"`
			} `json:"TimeSeries"`
		} `json:"MyEnergyData_MarketDocument"`
		Success    bool        `json:"success"`
		ErrorCode  int         `json:"errorCode"`
		ErrorText  string      `json:"errorText"`
		Id         string      `json:"id"`
		StackTrace interface{} `json:"stackTrace"`
	} `json:"result"`
}

// ElOverblikTokenResult is the result returned when calling /api/token
type ElOverblikTokenResult struct {
	Result string `json:"result"`
}

type EloverblikGetTimeSeriesRequest struct {
	MeteringPoints struct {
		MeteringPoint []string `json:"meteringPoint"`
	} `json:"meteringPoints"`
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
func (eo *ElOverblik) GetRequestToken(forceGetToken bool, saveToken bool) error {

	// check if Application token is configured, and not expired
	if eo.ApplicationToken.Token == "" {
		return errors.New("no Application token configured")
	}
	if eo.ApplicationToken.Expire.Before(time.Now()) {
		return errors.New("Application token has expired, please update it on eloverblik.dk website")
	}

	// if no request token is available, and saveToken is configured
	// then let's try and read the token from disk first
	if saveToken && eo.RequestToken.Token == "" {
		jsonToken, tokenExits, err := eo.ReadRequestTokenFromDisk()
		if err == nil {
			if tokenExits {
				log.Println("Read request token from disk")
				err = json.Unmarshal(jsonToken, &eo.RequestToken)
				if err != nil {
					log.Println("Unable to use the request token from disk:", err.Error())
				}
			}
		}
	}

	// check if the current request token has expired
	if eo.RequestToken.Token != "" && eo.RequestToken.Expire.After(time.Now()) && !forceGetToken {
		log.Println("The current request token is still valid")
		return nil
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

	// if configured let's save the request token to disk, there is a limitation on how many times
	// this application is allowed to request a token
	if saveToken {
		tokenJson, err := json.Marshal(&eo.RequestToken)
		if err != nil {
			log.Println("WARNING: unable to convert token to json, for storage:", err.Error())
		} else {
			err = eo.SaveRequestTokenToDisk(string(tokenJson))
			if err != nil {
				log.Println("WARNING: unable to save request token to disk:", err.Error())
			}
		}
	}

	return nil
}

// ReadRequestTokenFromDisk checks if there is a token located on disk, if so it returns it
func (eo *ElOverblik) ReadRequestTokenFromDisk() (tokenJson []byte, tokenExisted bool, err error) {
	// get application directory
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return tokenJson, false, err
	}

	// create the path
	filename := filepath.Join(dir, ".requestToken")

	// read the file
	tokenJson, err = os.ReadFile(filename)
	if err != nil {
		return tokenJson, false, err
	}

	return tokenJson, true, err

}

// SaveRequestTokenToDisk saves the provided token to .requestToken file
func (eo *ElOverblik) SaveRequestTokenToDisk(token string) error {
	// get application directory
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}

	// create the path
	filename := filepath.Join(dir, ".requestToken")

	// write the file
	err = os.WriteFile(filename, []byte(token), 0644)
	if err != nil {
		return err
	}
	return nil
}

// GetMeteringPoints get the meteringpoints for the token provided, and returns an array with the result
func (eo *ElOverblik) GetMeteringPoints() (Meteringpoints []EloverblikMeteringPoint, err error) {

	// create the context, timeout after 20 seconds
	timeoutContext, cancelFunc := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelFunc()

	// create the request
	req, err := http.NewRequestWithContext(timeoutContext, http.MethodGet, "https://api.eloverblik.dk/customerapi/api/meteringpoints/meteringpoints?includeAll=true", nil)
	if err != nil {
		return make([]EloverblikMeteringPoint, 0), err
	}

	// add the headers needed
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+eo.RequestToken.Token)

	// make the http request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return make([]EloverblikMeteringPoint, 0), err
	}

	// check the HTTP status code
	if res.StatusCode > 299 {
		return make([]EloverblikMeteringPoint, 0), errors.New("unable to get meteringpoints, server responded:" + res.Status)
	}

	// marshal the json result into struct
	mRes := EloverblikMeteringPointResult{}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return make([]EloverblikMeteringPoint, 0), err
	}

	// Unmarshal the json into the result struct
	err = json.Unmarshal(body, &mRes)
	if err != nil {
		return make([]EloverblikMeteringPoint, 0), err
	}

	// return the result
	return mRes.Result, nil
}

// GetMeterReadings  make the "gettimeseries" request towards eloverblik and returns the result
func (eo *ElOverblik) GetMeterReadings(meteringPoint string, fromDate time.Time, toDate time.Time) (result EloverblikMeteringTimeSeriesResult, err error) {

	// Get a string representation og the fromDate and toDate
	seriesFrom := fromDate.Format("2006-01-02")
	seriesTo := toDate.Format("2006-01-02")
	log.Println("Getting data fromDate:", seriesFrom, "toDate:", seriesTo)

	// create the context, timeout after 20 seconds
	timeoutContext, cancelFunc := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelFunc()

	// create the json for the body
	reqBody := EloverblikGetTimeSeriesRequest{}
	reqBody.MeteringPoints.MeteringPoint = []string{meteringPoint}
	bjson, err := json.Marshal(&reqBody)
	if err != nil {
		return result, err
	}

	// create the request
	req, err := http.NewRequestWithContext(timeoutContext, http.MethodPost, "https://api.eloverblik.dk/customerapi/api/meterdata/gettimeseries/"+seriesFrom+"/"+seriesTo+"/Hour", bytes.NewBuffer(bjson))
	if err != nil {
		return result, err
	}

	// add the headers needed
	req.Header.Add("accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+eo.RequestToken.Token)

	// make the http request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}

	// check the HTTP status code
	if res.StatusCode > 299 {
		return result, errors.New("unable to get meteringpoints, server responded:" + res.Status)
	}

	// marshal the json result into struct
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return result, err
	}

	// parse the json into the result struct
	err = json.Unmarshal(body, &result)
	if err != nil {
		return result, err
	}

	// return the data to the caller
	return result, nil
}
