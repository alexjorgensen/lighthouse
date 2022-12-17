package main

import (
	"database/sql"
	"fmt"
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
	db.handle, err = sql.Open("mysql", settings.Database.Username+":"+settings.Database.Password+"@tcp("+settings.Database.HostName+":3306)/lighthouse?parseTime=true")
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
			if rows.Close() != nil {
				fmt.Println("unable to close database rows.")
			}
		}
	}

	return nil
}

// SaveMeteringPoints saves each of the provided meteringpoints into to database
func (db *Database) SaveMeteringPoints(mps *[]EloverblikMeteringPoint) error {

	// insert each of the meteringspoints into database
	for _, mp := range *mps {
		// Prepare the INSERT statement
		var stmt, err = db.handle.Prepare("REPLACE INTO meteringPoint (streetCode, streetName, buildingNumber, floorId, roomId, citySubDivisionName, municipalityCode, locationDescription, settlementMethod, meterReadingOccurrence, firstConsumerPartyName, secondConsumerPartyName, meterNumber, consumerStartDate, meteringPointId, typeOfMp, balanceSupplierName, postcode, cityName, hasRelation) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		if err != nil {
			log.Fatal(err)
		}

		// Execute the INSERT statement
		_, err = stmt.Exec(
			mp.StreetCode,
			mp.StreetName,
			mp.BuildingNumber,
			mp.FloorId,
			mp.RoomId,
			mp.CitySubDivisionName,
			mp.MunicipalityCode,
			mp.LocationDescription,
			mp.SettlementMethod,
			mp.MeterReadingOccurrence,
			mp.FirstConsumerPartyName,
			mp.SecondConsumerPartyName,
			mp.MeterNumber,
			mp.ConsumerStartDate,
			mp.MeteringPointId,
			mp.TypeOfMP,
			mp.BalanceSupplierName,
			mp.Postcode,
			mp.CityName,
			mp.HasRelation,
		)
		if err != nil {
			return err
		}
		err = stmt.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// SameMeteringTimeSeries save each entry in the timeSeries slice to database
func (db *Database) SameMeteringTimeSeries(mts EloverblikMeteringTimeSeriesResult) error {
	for _, result := range mts.Result {
		for _, ts := range result.MyEnergyDataMarketDocument.TimeSeries {
			for _, p := range ts.Period {
				for _, point := range p.Point {
					// Prepare the INSERT statement
					var stmt, err = db.handle.Prepare("REPLACE INTO meteringPointsTimeSeries (meteringPointId,measurementUnit,businessType,hour,quantity,quality) VALUES (?,?,?,?,?,?)")
					if err != nil {
						return err
					}
					// Execute the INSERT statement
					pos, err := strconv.Atoi(point.Position)
					if err != nil {
						return err
					}
					cHour := p.TimeInterval.Start.Add(time.Duration(pos) * time.Hour)
					_, err = stmt.Exec(ts.MRID, ts.MeasurementUnitName, ts.BusinessType, cHour, point.OutQuantityQuantity, point.OutQuantityQuality)
					if err != nil {
						log.Fatal(err)
						return err
					}
					err = stmt.Close()
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
