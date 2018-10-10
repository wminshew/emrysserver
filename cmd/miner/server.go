// package main begins a miner server
package main

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/wminshew/emrys/pkg/validate"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/auth"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"github.com/wminshew/emrysserver/pkg/storage"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	rMinerAuth.Handle("/device_snapshot", postDeviceSnapshot()).Methods("POST")
	rMinerAuth.Use(auth.Jwt(minerSecret))

	uuidRegexpMux := validate.UUIDRegexpMux()
	rMinerJob := rMinerAuth.PathPrefix(fmt.Sprintf("/job/{jID:%s}", uuidRegexpMux)).Subrouter()
	rMinerJob.Handle("/bid", postBid()).Methods("POST")
	rMinerJob.Use(auth.JobActive())

	rAuction := r.PathPrefix("/auction").Subrouter()
	rAuction.Handle(fmt.Sprintf("/{jID:%s}", uuidRegexpMux), postAuction()).Methods("POST")
	rAuction.Use(auth.Jwt(userSecret))
	rAuction.Use(auth.UserJobMiddleware())
	rAuction.Use(auth.JobActive())

	server := http.Server{
		Addr:              ":8080",
		Handler:           log.Log(r),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Sugar.Infof("Miner listening on port %s...", server.Addr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Sugar.Fatalf("Server error: %v", err)
		}
	}()

	ctx := context.Background()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop
	if err := server.Shutdown(ctx); err != nil {
		log.Sugar.Errorf("shutting server down: %v", err)
	}
}
