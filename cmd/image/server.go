// package main begins an image server
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
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	userSecret       = os.Getenv("USERSECRET")
	minerSecret      = os.Getenv("MINERSECRET")
	registryHost     = os.Getenv("REGISTRY_HOST")
	devpiHost        = os.Getenv("DEVPI_HOST")
	devpiTrustedHost string
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
	initDocker()
	defer func() {
		if err := dClient.Close(); err != nil {
			log.Sugar.Errorf("Error closing docker client: %v\n", err)
		}
	}()
	if u, err := url.Parse(devpiHost); err != nil {
		log.Sugar.Errorf("Error parsing devpiHost %s: %v\n", devpiHost, err)
		panic(err)
	} else {
		devpiTrustedHost = u.Hostname()
	}

	r := mux.NewRouter()
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rImage := r.PathPrefix("/image").HeadersRegexp("Authorization", "^Bearer ").Methods("POST").Subrouter()

	rImageMiner := rImage.PathPrefix("/downloaded").Subrouter()
	rImageMiner.Use(auth.Jwt(minerSecret))
	rImageMiner.Use(auth.MinerJobMiddleware())
	rImageMiner.Use(auth.JobActive())
	uuidRegexpMux := validate.UUIDRegexpMux()
	rImageMiner.Handle(fmt.Sprintf("/{jID:%s}", uuidRegexpMux), imageDownloaded())

	projectRegexpMux := validate.ProjectRegexpMux()
	rImageUser := rImage.PathPrefix(fmt.Sprintf("/{uID:%s}/{project:%s}", uuidRegexpMux, projectRegexpMux)).Subrouter()
	rImageUser.Use(auth.Jwt(userSecret))
	rImageUser.Use(auth.UserJobMiddleware())
	rImageUser.Use(auth.JobActive())
	rImageUser.Handle(fmt.Sprintf("/{jID:%s}", uuidRegexpMux), buildImage())

	server := http.Server{
		Addr:              ":8080",
		Handler:           log.Log(r),
		ReadHeaderTimeout: 15 * time.Second,
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
