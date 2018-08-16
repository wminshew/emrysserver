// package main begins an image server
package main

import (
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
	// "time"
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

	rImageUser := r.PathPrefix("/image").HeadersRegexp("Authorization", "^Bearer ").Methods("POST").Subrouter()
	projectRegexpMux := validate.ProjectRegexpMux()
	buildImagePath := fmt.Sprintf("/{uID}/{project:%s}/{jID}", projectRegexpMux)
	rImageUser.Handle(buildImagePath, buildImage())
	rImageUser.Use(auth.Jwt(userSecret))
	rImageUser.Use(auth.UserJobMiddleware())

	server := http.Server{
		Addr:    ":8080",
		Handler: log.Log(r),
		// ReadHeaderTimeout: 5 * time.Second,
	}

	log.Sugar.Infof("Listening on port %s...", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Sugar.Fatalf("Server error: %v", err)
	}
}
