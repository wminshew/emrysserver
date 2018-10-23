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
	"github.com/wminshew/emrysserver/pkg/payments"
	"github.com/wminshew/emrysserver/pkg/storage"
	"gopkg.in/robfig/cron.v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var minerSecret = os.Getenv("MINER_SECRET")
var userSecret = os.Getenv("USER_SECRET")
var sendgridSecret = os.Getenv("SENDGRID_SECRET")

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
	c := cron.New()
	defer c.Stop()
	if _, err := c.AddFunc("@weekly", payments.PayMiners); err != nil {
		log.Sugar.Errorf("Error starting weekly miner payment to cron: %v", err)
		panic(err)
	}

	uuidRegexpMux := validate.UUIDRegexpMux()

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rMiner := r.PathPrefix("/miner").Subrouter()
	rMiner.Handle("", newMiner).Methods("POST")
	rMiner.Handle("/confirm", confirmMiner).Methods("GET")
	rMiner.Handle("/login", login).Methods("POST")
	rMiner.Handle("/version", getVersion).Methods("GET")

	rMinerAuth := rMiner.NewRoute().HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	rMinerAuth.Use(auth.Jwt(minerSecret))
	rMinerAuth.Handle("/connect", auth.MinerActive(connect)).Methods("GET")
	rMinerAuth.Handle("/device_snapshot", postDeviceSnapshot).Methods("POST")
	postBidPath := fmt.Sprintf("/job/{jID:%s}/bid", uuidRegexpMux)
	rMinerAuth.Handle(postBidPath, auth.JobActive(postBid)).Methods("POST")

	rAuction := r.PathPrefix("/auction").Subrouter()
	rAuction.Use(auth.Jwt(userSecret))
	rAuction.Use(auth.UserJobMiddleware)
	rAuction.Use(auth.JobActive)
	postAuctionPath := fmt.Sprintf("/{jID:%s}", uuidRegexpMux)
	rAuction.Handle(postAuctionPath, postAuction).Methods("POST")

	server := http.Server{
		Addr:              ":8080",
		Handler:           log.Log(r),
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       620 * time.Second, // per https://cloud.google.com/load-balancing/docs/https/#timeouts_and_retries
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
