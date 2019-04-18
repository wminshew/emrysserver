package db

import (
	"database/sql"
	"fmt"
	// psql driver
	_ "github.com/lib/pq"
	"github.com/wminshew/emrysserver/pkg/log"
	"os"
	"strconv"
	"time"
)

var (
	dbUser               = os.Getenv("DB_USER")
	dbPassword           = os.Getenv("DB_PASSWORD")
	dbNetloc             = os.Getenv("DB_NETLOC")
	dbPort               = os.Getenv("DB_PORT")
	dbName               = os.Getenv("DB_NAME")
	dbMaxOpenConnsStr    = os.Getenv("DBMAXOPENCONNS")
	dbMaxOpenConns       = 10
	dbMaxIdleConnsStr    = os.Getenv("DBMAXIDLECONNS")
	dbMaxIdleConns       = 5
	dbMaxConnLifetimeStr = os.Getenv("DBCONNMAXLIFETIME")
	dbMaxConnLifetime    = 10 * time.Minute
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

	if dbMaxOpenConnsStr != "" {
		dbMaxOpenConns, err = strconv.Atoi(dbMaxOpenConnsStr)
		if err != nil {
			log.Sugar.Errorf("Error converting max_open_conns: %v", err)
			panic(err)
		}
	}
	db.SetMaxOpenConns(dbMaxOpenConns)

	if dbMaxIdleConnsStr != "" {
		dbMaxIdleConns, err = strconv.Atoi(dbMaxIdleConnsStr)
		if err != nil {
			log.Sugar.Errorf("Error converting max_idle_conns: %v", err)
			panic(err)
		}
	}
	db.SetMaxIdleConns(dbMaxIdleConns)

	if dbMaxConnLifetimeStr != "" {
		dbMaxConnLifetimeMin, err := strconv.Atoi(dbMaxConnLifetimeStr)
		if err != nil {
			log.Sugar.Errorf("Error converting max_conn_lifetime: %v", err)
			panic(err)
		}

		dbMaxConnLifetime = time.Duration(dbMaxConnLifetimeMin) * time.Minute
	}
	db.SetConnMaxLifetime(dbMaxConnLifetime)
}

// Close closes the database connection
func Close() {
	log.Sugar.Infof("Closing database...")

	if err := db.Close(); err != nil {
		log.Sugar.Errorf("Error closing database: %v", err)
	}
}
