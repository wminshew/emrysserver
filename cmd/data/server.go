// package main begins a data server
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
	"os"
	"path"
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
	go func() {
		for {
			if err := startDiskManager(); err != nil {
				log.Sugar.Errorf("Error managing disk utilization: %v\n", err)
			}
		}
	}()

	r := mux.NewRouter()
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rDataUser := r.PathPrefix("/user").HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	projectRegexpMux := validate.ProjectRegexpMux()
	syncUserPath := fmt.Sprintf("/{uID}/project/{project:%s}/job/{jID}", projectRegexpMux)
	rDataUser.Handle(syncUserPath, syncUser()).Methods("POST")
	uploadDataPath := path.Join(syncUserPath, "{relPath:.*}")
	rDataUser.Handle(uploadDataPath, uploadData()).Methods("PUT")
	rDataUser.Use(auth.Jwt(userSecret))
	rDataUser.Use(auth.UserJobMiddleware())

	rDataMiner := r.PathPrefix("/miner").HeadersRegexp("Authorization", "^Bearer ").Methods("GET").Subrouter()
	rDataMiner.Handle("/job/{jID}", getData())
	rDataMiner.Use(auth.Jwt(minerSecret))
	rDataMiner.Use(auth.MinerJobMiddleware())

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
