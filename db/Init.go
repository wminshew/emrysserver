package db

import (
	"database/sql"
	"fmt"
	// psql driver
	_ "github.com/lib/pq"
	"github.com/wminshew/emrysserver/pkg/app"
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

// Init initializes the database
func Init() {
	app.Sugar.Infof("Initializing database...")

	var err error
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbNetloc, dbPort, dbName)
	if Db, err = sql.Open("postgres", connStr); err != nil {
		app.Sugar.Errorf("Error connecting to database: %v", err)
		panic(err)
	}
}
