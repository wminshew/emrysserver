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
	"time"
)

var userSecret = os.Getenv("USER_SECRET")
var minerSecret = os.Getenv("MINER_SECRET")

func main() {
	log.Init()
	defer func() {
		if err := log.Sugar.Sync(); err != nil {
			log.Sugar.Errorf("Error syncing log: %v\n", err)
		}
	}()
	db.Init()
	defer db.Close()

	uuidRegexpMux := validate.UUIDRegexpMux()
	projectRegexpMux := validate.ProjectRegexpMux()

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rUser := r.PathPrefix("/user").Subrouter()
	rUser.Handle("", newUser).Methods("POST")
	rUser.Handle("/confirm", confirmUser).Methods("GET")
	rUser.Handle("/login", login).Methods("POST")
	rUser.Handle("/version", getVersion).Methods("GET")

	jobPathPrefix := fmt.Sprintf("/{uID:%s}/project/{project:%s}/job", uuidRegexpMux, projectRegexpMux)
	rUserAuth := rUser.PathPrefix(jobPathPrefix).HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	rUserAuth.Use(auth.Jwt(userSecret))
	rUserAuth.Handle("", auth.UserActive(postJob)).Methods("POST")
	postCancelPath := fmt.Sprintf("/{jID:%s}/cancel", uuidRegexpMux)
	rUserAuth.Handle(postCancelPath,
		auth.JobActive(auth.UserJobMiddleware(postCancelJob))).Methods("POST")

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
	if err := server.Shutdown(ctx); err != nil {
		log.Sugar.Errorf("shutting server down: %v", err)
	}
}
