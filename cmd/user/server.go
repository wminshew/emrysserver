// package main begins a user server
package main

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/auth"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"os"
)

var userSecret = os.Getenv("USERSECRET")
var minerSecret = os.Getenv("MINERSECRET")

func main() {
	log.Init()
	defer func() {
		err := log.Sugar.Sync()
		log.Sugar.Errorf("Error syncing log: %v\n", err)
	}()
	db.Init()
	defer db.Close()
	initStorage()
	initDocker()
	defer func() {
		err := dClient.Close()
		log.Sugar.Errorf("Error closing docker client: %v\n", err)
	}()

	r := mux.NewRouter()
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rUser := r.PathPrefix("/user").Subrouter()
	rUser.Handle("", newUser()).Methods("POST")
	rUser.Handle("/", newUser()).Methods("POST")
	rUser.Handle("/login", login()).Methods("POST")
	rUser.Handle("/version", getVersion()).Methods("GET")

	rUserAuth := rUser.NewRoute().HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	rUserAuth.Handle("/{uID}/job", postJob()).Methods("POST")
	rUserAuth.Use(auth.Jwt(userSecret))

	rImage := r.PathPrefix("/image").HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	rImageUser := rImage.Methods("POST").Subrouter()
	rImageUser.Handle("/{jID}", buildImage())
	rImageUser.Use(auth.Jwt(userSecret))
	rImageUser.Use(auth.UserJobMiddleware())

	// rImageMiner := rImage.Methods("GET").Subrouter()
	// rImageMiner.Handle("/{jID}", buildImage())
	// rImageMiner.Use(auth.Jwt(minerSecret))
	// rImageMiner.Use(auth.MinerJobMiddleware())

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
