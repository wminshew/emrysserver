// package main begins a job server
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
)

var userSecret = os.Getenv("USERSECRET")
var minerSecret = os.Getenv("MINERSECRET")

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

	rJob := r.PathPrefix("/job").Subrouter()

	rJobMiner := rJob.NewRoute().Methods("POST").HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	rJobMiner.Handle("/{jID}/log", postOutputLog())
	rJobMiner.Handle("/{jID}/data", postOutputData())
	rJobMiner.Use(auth.Jwt(minerSecret))
	rJobMiner.Use(auth.MinerJobMiddleware())

	rJobUser := rJob.NewRoute().Methods("GET").HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	rJobUser.Handle("/{jID}/log", getOutputLog())
	rJobUser.Handle("/{jID}/data", getOutputData())
	rJobUser.Use(auth.Jwt(userSecret))
	rJobUser.Use(auth.UserJobMiddleware())

	server := http.Server{
		Addr:    ":8080",
		Handler: log.Log(r),
	}

	log.Sugar.Infof("Listening on port %s...", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Sugar.Fatalf("Server error: %v", err)
	}
}
