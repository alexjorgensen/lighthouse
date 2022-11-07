package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {

	// connect to database
	/*	db := Database{}
		err := db.ConnectToDatabase()
		if err != nil {
			fmt.Println("error connecting to db:", err.Error())
			os.Exit(1)
		}*/

	// Read configuration
	settings := Settings{}
	err := settings.ReadConfigurationFile()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	n := NorlysAPI{}
	prices, err := n.GetPrices(1, &settings)
	if err != nil {
		fmt.Println("Error getting prices from norlys:", err.Error())
		os.Exit(1)
	}

	count := 0
	total := 0.0
	for _, pd := range prices {
		for _, p := range pd.DisplayPrices {
			// try and convert the timestamp string to int
			t, err := strconv.Atoi(p.Time)
			if err == nil {

				d := pd.PriceDate.Add(time.Duration(t) * time.Hour)
				total += p.Value
				fmt.Println(d, p.Value)
				count++
			}
		}
	}
	fmt.Println("total:", total, ", count:", count, " EQ:", total/float64(count))
}
