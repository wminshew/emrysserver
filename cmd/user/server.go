// package main begins a user server
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
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	r := mux.NewRouter()
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rUser := r.PathPrefix("/user").Subrouter()
	rUser.Handle("", newUser()).Methods("POST")
	rUser.Handle("/", newUser()).Methods("POST")
	rUser.Handle("/login", login()).Methods("POST")
	rUser.Handle("/version", getVersion()).Methods("GET")

	rUserAuth := rUser.NewRoute().HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	projectRegexpMux := validate.ProjectRegexpMux()
	postJobPath := fmt.Sprintf("/{uID}/project/{project:%s}/job", projectRegexpMux)
	rUserAuth.Handle(postJobPath, postJob()).Methods("POST")
	rUserAuth.Use(auth.Jwt(userSecret))

	server := http.Server{
		Addr:    ":8080",
		Handler: log.Log(r),
		// ReadHeaderTimeout: 5 * time.Second,
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
