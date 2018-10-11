// package main begins a data server
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
	"path"
	"syscall"
	"time"
)

var (
	userSecret  = os.Getenv("USERSECRET")
	minerSecret = os.Getenv("MINERSECRET")
)

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
	initMetadataSync()
	initDiskManager()
	done := make(chan struct{}, 1)
	go func() {
		for {
			if err := checkAndEvictProjects(); err != nil {
				log.Sugar.Errorf("Error managing disk utilization: %v\n", err)
			}
			select {
			case <-done:
				return
			case <-time.After(time.Duration(pvcPeriodSec) * time.Second):
			}
		}
	}()

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rDataUser := r.PathPrefix("/user").HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	uuidRegexpMux := validate.UUIDRegexpMux()
	projectRegexpMux := validate.ProjectRegexpMux()
	syncUserPath := fmt.Sprintf("/{uID:%s}/project/{project:%s}/job/{jID}", uuidRegexpMux, projectRegexpMux)
	rDataUser.Handle(syncUserPath, syncUser()).Methods("POST")
	rDataUser.Handle(path.Join(syncUserPath, "{relPath:.*}"), uploadData()).Methods("PUT")
	rDataUser.Use(auth.Jwt(userSecret))
	rDataUser.Use(auth.UserJobMiddleware())
	rDataUser.Use(auth.JobActive())

	rDataMiner := r.PathPrefix("/miner").HeadersRegexp("Authorization", "^Bearer ").Methods("GET").Subrouter()
	rDataMiner.Handle(fmt.Sprintf("/job/{jID:%s}", uuidRegexpMux), getData())
	rDataMiner.Use(auth.Jwt(minerSecret))
	rDataMiner.Use(auth.MinerJobMiddleware())
	rDataMiner.Use(auth.JobActive())

	server := http.Server{
		Addr:              ":8080",
		Handler:           log.Log(r),
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       620 * time.Second, // per https://cloud.google.com/load-balancing/docs/https/#timeouts_and_retries
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
	close(done)
	if err := server.Shutdown(ctx); err != nil {
		log.Sugar.Errorf("shutting server down: %v", err)
	}
}
