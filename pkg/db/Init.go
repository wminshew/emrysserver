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

var db *sql.DB

// Init initializes the database connection
func Init() {
	log.Sugar.Infof("Initializing database...")

	var err error
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbNetloc, dbPort, dbName)
	if db, err = sql.Open("postgres", connStr); err != nil {
		log.Sugar.Errorf("Error opening database: %v", err)
		panic(err)
	}

	if err = db.Ping(); err != nil {
		log.Sugar.Errorf("Error pinging database: %v", err)
		panic(err)
	}
}

// Close closes the database connection
func Close() {
	log.Sugar.Infof("Closing database...")

	if err := db.Close(); err != nil {
		log.Sugar.Errorf("Error closing database: %v", err)
	}
}
