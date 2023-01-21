package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"os"
	"strconv"
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
		log.Println("error connecting to db:", err.Error())
		os.Exit(1)
	}

	// Manage updating and saving of Norlys prices
	go GetAndSaveNorlysPrices(&settings, &db)

	// Manage updating and saving of Eloverblik Data
	go GetAndSaveEloverblikData(&settings, &db)

	// init the echo library
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.GET("/usage", HandleGETUsage)
	log.Println("Listening for HTTPS requests on port 4001")
	if err := e.Start("::" + strconv.Itoa(settings.APIPort)); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
