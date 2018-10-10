// package main begins a job server
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

	uuidRegexpMux := validate.UUIDRegexpMux()
	rJob := r.PathPrefix(fmt.Sprintf("/job/{jID:%s}", uuidRegexpMux)).Subrouter()

	rJobMiner := rJob.NewRoute().Methods("POST").HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	rJobMiner.Handle("/log", postOutputLog())
	rJobMiner.Handle("/data", postOutputData())
	rJobMiner.Use(auth.Jwt(minerSecret))
	rJobMiner.Use(auth.MinerJobMiddleware())
	rJobMiner.Use(auth.JobActive())

	rJobUser := rJob.NewRoute().Methods("GET").HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	rJobUser.Handle("/log", streamOutputLog())
	rJobUser.Handle("/log/download", downloadOutputLog())
	rJobUser.Handle("/data", getOutputData())
	rJobUser.Use(auth.Jwt(userSecret))
	rJobUser.Use(auth.UserJobMiddleware())

	server := http.Server{
		Addr:              ":8080",
		Handler:           log.Log(r),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Sugar.Infof("Listening on port %s...", server.Addr)
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
