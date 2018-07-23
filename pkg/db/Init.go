package db

import (
	"database/sql"
	"fmt"
	// psql driver
	_ "github.com/lib/pq"
	"github.com/wminshew/emrysserver/pkg/log"
	"os"
)

var (
	dbUser     = os.Getenv("DBUSER")
	dbPassword = os.Getenv("DBPASSWORD")
	dbNetloc   = os.Getenv("DBNETLOC")
	dbPort     = os.Getenv("DBPORT")
	dbName     = os.Getenv("DBNAME")
)

// Db is the database
var Db *sql.DB

// Init initializes the database connection
func Init() {
	log.Sugar.Infof("Initializing database...")

	var err error
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbNetloc, dbPort, dbName)
	if Db, err = sql.Open("postgres", connStr); err != nil {
		log.Sugar.Errorf("Error opening database: %v", err)
		panic(err)
	}

	if err = Db.Ping(); err != nil {
		log.Sugar.Errorf("Error pinging database: %v", err)
		panic(err)
	}
}

// Close closes the database connection
func Close() {
	log.Sugar.Infof("Closing database...")

	if err := Db.Close(); err != nil {
		log.Sugar.Errorf("Error closing database: %v", err)
	}
}
