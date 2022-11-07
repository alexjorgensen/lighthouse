package main

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

// Database is used to connect and execute database queries
type Database struct {
	handle *sql.DB
}

// ConnectToDatabase connects to database
func (db *Database) ConnectToDatabase() error {
	var err error
	db.handle, err = sql.Open("mysql", "grafana:opsxirut&ri2wE!@tcp(db:3306)/thue")
	if err != nil {
		log.Println("ERROR connecting to mysql database:", err.Error())
		return err
	}
	return nil
}
