// package main begins an image server
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
	// "time"
)

var (
	userSecret   = os.Getenv("USERSECRET")
	minerSecret  = os.Getenv("MINERSECRET")
	registryHost = os.Getenv("REGISTRY_HOST")
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

	r := mux.NewRouter()
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rImageUser := r.PathPrefix("/image").HeadersRegexp("Authorization", "^Bearer ").Methods("POST").Subrouter()
	rImageUser.Handle("/{jID}", buildImage())
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
