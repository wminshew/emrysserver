// package main begins a miner server
package main

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/auth"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"github.com/wminshew/emrysserver/pkg/storage"
	"net/http"
	"os"
	"time"
)

var minerSecret = os.Getenv("MINERSECRET")
var userSecret = os.Getenv("USERSECRET")

func main() {
	log.Init()
	defer func() {
		if err := log.Sugar.Sync(); err != nil {
			log.Sugar.Errorf("Error syncing log: %v\n", err)
		}
	}()
	db.Init()
	defer db.Close()
	storage.Init()
	initJobsManager()

	r := mux.NewRouter()
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rMiner := r.PathPrefix("/miner").Subrouter()
	rMiner.Handle("", newMiner()).Methods("POST")
	rMiner.Handle("/", newMiner()).Methods("POST")
	rMiner.Handle("/login", login()).Methods("POST")
	rMiner.Handle("/version", getVersion()).Methods("GET")

	rMinerAuth := rMiner.NewRoute().HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	rMinerAuth.Handle("/connect", connect()).Methods("GET")
	rMinerAuth.Handle("/job/{jID}/bid", postBid()).Methods("POST")
	rMinerAuth.Use(auth.Jwt(minerSecret))

	rAuction := r.PathPrefix("/auction").Subrouter()
	rAuction.Handle("/{jID}", postAuction()).Methods("POST")
	rAuction.Use(auth.Jwt(userSecret))
	rAuction.Use(auth.UserJobMiddleware())

	server := http.Server{
		Addr:        ":8080",
		Handler:     log.Log(r),
		ReadTimeout: 5 * time.Second,
	}

	log.Sugar.Infof("Miner listening on port %s...", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Sugar.Fatalf("Server error: %v", err)
	}
}
