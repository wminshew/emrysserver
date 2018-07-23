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

var secret = os.Getenv("USERSECRET")

func main() {
	log.Init()
	defer func() { _ = log.Sugar.Sync() }()
	db.Init()
	defer db.Close()
	initStorage()
	initDocker()

	r := mux.NewRouter()
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rUser := r.PathPrefix("/user").Subrouter()
	rUser.Handle("", newUser()).Methods("POST")
	rUser.Handle("/", newUser()).Methods("POST")
	rUser.Handle("/login", login()).Methods("POST")
	rUser.Handle("/version", getVersion()).Methods("GET")
	rUser.Handle("/{uID}/job", postJob()).Methods("POST").
		HeadersRegexp("Authorization", "^Bearer ").
		Subrouter().Use(auth.Jwt(secret))

	rImage := r.PathPrefix("/image").HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	rImage.Handle("/{jID}", buildImage()).Methods("POST")
	rImage.Use(auth.Jwt(secret))
	rImage.Use(auth.UserJobMiddleware())

	server := http.Server{
		Addr:    ":8080",
		Handler: log.Log(r),
	}

	log.Sugar.Infof("Listening on port %s...", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Sugar.Fatalf("Server error: %v", err)
	}
}
