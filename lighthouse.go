package main

import (
	"log"
	"os"
	"time"
)

// GetAndSaveNorlysPrices fetches data from norlys and saves it to database
func GetAndSaveNorlysPrices(settings *Settings, db *Database) {
	n := NorlysAPI{}
	for {
		// get the current norlys prices, and update the database
		log.Println("Getting prices from Norlys...")
		prices, err := n.GetPrices(settings.NumberOfDaysForPrices, settings)
		if err != nil {
			// we got an error while trying to get the prices from Norlys, we'll wait 60 seconds and try again.
			log.Println("Error getting prices from norlys:", err.Error())
			time.Sleep(60 * time.Second)
			continue
		}

		log.Println("Saving prices to database")
		for _, pd := range prices {
			err = db.SaveNorlysPricingResult(&pd)
			if err != nil {
				log.Println("Error saving the prices to db:", err.Error())
			}
		}

		// wait until the configured time has passed before updating the DB again
		time.Sleep(time.Duration(settings.NorlysAPI.UpdatePricesInterval) * time.Second)
	}
}

// GetAndSaveEloverblikData fetches all data from eloverblik an saves it to database
func GetAndSaveEloverblikData(settings *Settings, db *Database) {
	eo := ElOverblik{}

	// set the application token
	err := eo.SetApplicationToken(settings.ElOverblik.LighthouseToken)
	if err != nil {
		log.Println("ERROR:", err.Error())
		os.Exit(1)
	}

	for {
		// let's make a token request to get a request token
		log.Println("Getting request token from Eloverblik")
		err := eo.GetRequestToken(false, settings.SaveRequestTokenToDisk)
		if err != nil {
			log.Println("Error getting request token from eloverblik:", err.Error())
			time.Sleep(60 * time.Second)
			continue
		}

		// let's get the meteringspoints associated to the account
		log.Println("Getting meteringpoints from Eloverblik")
		mps, err := eo.GetMeteringPoints()
		if err != nil {
			log.Println("Error getting meteringpoints from eloverblik:", err.Error())
			time.Sleep(60 * time.Second)
			continue
		}

		// let's save the meteringpoints to database
		eo.MeteringPoints = mps
		log.Println("Saving Eloverblik data to database")
		err = db.SaveMeteringPoints(&mps)
		if err != nil {
			log.Println("Error saving meteringpoints to database:", err.Error())
			time.Sleep(60 * time.Second)
			continue
		}

		for _, mp := range mps {
			// let's get the latest time-series data associated to this meteringpoint
			fromDate := time.Now().Add(-time.Hour * time.Duration(settings.NumberOfDaysForMeteringData*24))
			toDate := time.Now().Add(-time.Hour * 1)
			meterReadings, err := eo.GetMeterReadings(mp.MeteringPointId, fromDate, toDate)
			if err != nil {
				log.Println("Error getting meter time-series data:", err.Error())
				time.Sleep(60 * time.Second)
				continue
			}

			if len(meterReadings.Result) > 0 {
				// let's save the data to database
				err = db.SaveMeteringTimeSeries(meterReadings)
				if err != nil {
					log.Println("Error saving meter time-series data to db:", err.Error())
					time.Sleep(60 * time.Second)
					continue
				}
			}
		}

		log.Println("Done fetching data from Eloverblik")

		// wait until the configured time has passed before updating the DB again
		time.Sleep(time.Duration(settings.NorlysAPI.UpdatePricesInterval) * time.Second)
	}
}
