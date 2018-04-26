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

var Db *sql.DB

func Init() {
	var err error

	// postgresql://[user[:password]@][netloc][:port][,...][/dbname][?param1=value1&...]
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbNetloc, dbPort, dbName)
	Db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
}
