// TODO: should this be moved to package main?
package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"os"
)

var (
	dbUser     = os.Getenv("DBUSER")
	dbPassword = os.Getenv("DBPASSWORD")
	dbNetloc   = os.Getenv("DBNETLOC")
	dbPort     = os.Getenv("DBPORT")
	dbName     = os.Getenv("DBNAME")
)

// TODO: db.Db is not a good name; think more here. Could just put in package main.
// I guess eventually we won't export db -- just methods on db so
// this will be fine.....
var Db *sql.DB

func Init() {
	var err error

	// postgresql://[user[:password]@][netloc][:port][,...][/dbname][?param1=value1&...]
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbNetloc, dbPort, dbName)
	// connStr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", dbUser, dbPassword, dbName)
	Db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
}
