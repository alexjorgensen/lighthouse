package main

import (
	"database/sql"
	"log"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Database is used to connect and execute database queries
type Database struct {
	handle *sql.DB
}

// ConnectToDatabase connects to database
func (db *Database) ConnectToDatabase(settings *Settings) error {
	var err error
	db.handle, err = sql.Open("mysql", settings.Database.Username+":"+settings.Database.Password+"@tcp("+settings.Database.HostName+":3306)/lighthouse")
	if err != nil {
		log.Println("ERROR connecting to mysql database:", err.Error())
		return err
	}
	return nil
}

// SaveNorlysPricingResult saves the norlys pricedata to database
func (db *Database) SaveNorlysPricingResult(pd *NorlysPricingResult) error {

	// Insert a record into the database for each hour
	for _, p := range pd.DisplayPrices {
		// try and convert the timestamp string to int
		t, err := strconv.Atoi(p.Time)
		if err == nil {
			d := pd.PriceDate.Add(time.Duration(t) * time.Hour)

			SQL := "REPLACE INTO priceData (priceDate,sector,currency,hour,price) VALUES (?,?,?,?,?)"
			rows, err := db.handle.Query(SQL, pd.PriceDate, pd.Sector, pd.Currency, d, p.Value)
			if err != nil {
				return err
			}
			rows.Close()
		}
	}

	return nil
}
