// TODO: should this be moved to package main?
package db

import (
	"database/sql"
	_ "github.com/lib/pq"
	// "log"
)

// TODO: db.db is not a good name; think more here
// I guess eventually we won't export db -- just methods on db so
// this will be fine.....
var Db *sql.DB

func Init() {
	var err error

	// TODO: make sure
	// connStr := "postgres://pqgotest:password@localhost/pqgotest?sslmode=verify-full"
	connStr := "user=emrysuser password=simplepassword dbname=emrysuser sslmode=disable"
	Db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
	// err = Db.Ping()
	// if err != nil {
	// 	panic(err)
	// }
	// log.Printf(string(Db.Stats().OpenConnections))
}
