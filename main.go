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

	// Manage updating and saving of Norlys prices
	go GetAndSaveNorlysPrices(&settings, &db)

	// Manage updating and saving of Eloverblik Data
	go GetAndSaveEloverblikData(&settings, &db)

	for {
		time.Sleep(1 * time.Hour)
	}
}
