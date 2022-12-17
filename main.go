package main

import (
	"fmt"
	"os"
	"time"
)

func main() {

	// Read configuration file
	settings := Settings{}
	err := settings.ReadConfigurationFile()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// connect to database
	db := Database{}
	err = db.ConnectToDatabase(&settings)
	if err != nil {
		fmt.Println("error connecting to db:", err.Error())
		os.Exit(1)
	}

	n := NorlysAPI{}
	eo := ElOverblik{}

	numberOfDaysForPrices := 2
	numberOfDaysForMeteringData := 10

	// set the application token
	if eo.SetApplicationToken(settings.ElOverblik.LighthouseToken) != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}

	for {
		// get the current norlys prices, and update the database
		fmt.Println("Gettings prices from Norlys...")
		prices, err := n.GetPrices(numberOfDaysForPrices, &settings)
		if err != nil {
			// we got an error while trying to get the prices from Norlys, we'll wait 60 seconds and try again.
			fmt.Println("Error getting prices from norlys:", err.Error())
			time.Sleep(60 * time.Second)
			continue
		}

		fmt.Println("Saving prices to database")
		for _, pd := range prices {
			err = db.SaveNorlysPricingResult(&pd)
			if err != nil {
				fmt.Println("Error saving the prices to db:", err.Error())
			}
		}

		// let's make a token request to get a request token
		fmt.Println("Getting request token from Eloverblik")
		err = eo.GetRequestToken(false, settings.SaveRequestTokenToDisk)
		if err != nil {
			fmt.Println("Error getting request token from eloverblik:", err.Error())
			time.Sleep(60 * time.Second)
			continue
		}

		// let's get the meteringspoints associated to the account
		fmt.Println("Getting meteringpoints from Eloverblik")
		mps, err := eo.GetMeteringPoints()
		if err != nil {
			fmt.Println("Error getting meteringpoints from eloverblik:", err.Error())
			time.Sleep(60 * time.Second)
			continue
		}

		// let's save the meteringpoints to database
		eo.MeteringPoints = mps
		err = db.SaveMeteringPoints(&mps)
		if err != nil {
			fmt.Println("Error saving meteringpoints to database:", err.Error())
			time.Sleep(60 * time.Second)
			continue
		}

		for _, mp := range mps {
			// let's get the latest time-series data associated to this meteringpoint
			fromDate := time.Now().Add(-time.Hour * time.Duration(numberOfDaysForMeteringData*24))
			toDate := time.Now().Add(-time.Hour * 1)
			meterReadings, err := eo.GetMeterReadings(mp.MeteringPointId, fromDate, toDate)
			if err != nil {
				fmt.Println("Error getting meter time-series data:", err.Error())
				time.Sleep(60 * time.Second)
				continue
			}

			if len(meterReadings.Result) > 0 {
				// let's save the data to database
				err = db.SameMeteringTimeSeries(meterReadings)
				if err != nil {
					fmt.Println("Error saving meter time-series data to db:", err.Error())
					time.Sleep(60 * time.Second)
					continue
				}
			}
		}

		fmt.Println("All done")

		// wait until the configured time has passed before updating the DB again
		time.Sleep(time.Duration(settings.NorlysAPI.UpdatePricesInterval) * time.Second)
	}
}
