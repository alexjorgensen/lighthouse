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

	for {
		// get the current norlys prices, and update the database
		fmt.Println("Gettings prices from Norlys...")
		prices, err := n.GetPrices(1, &settings)
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

		// wait until the configured time has passed befor updating the DB again
		time.Sleep(time.Duration(settings.NorlysAPI.UpdatePricesInterval) * time.Second)
	}
}
