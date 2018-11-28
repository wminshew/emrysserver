// package main begins a user server
package main

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/wminshew/emrys/pkg/validate"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/auth"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"github.com/wminshew/emrysserver/pkg/payments"
	cronPkg "gopkg.in/robfig/cron.v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	authSecret = os.Getenv("AUTH_SECRET")
	debugCors  = (os.Getenv("DEBUG_CORS") == "true")
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
	cron := cronPkg.New()
	defer cron.Stop()
	if _, err := cron.AddFunc("@weekly", payments.AccountsPayout); err != nil {
		log.Sugar.Errorf("Error adding weekly payments to cron: %v", err)
		panic(err)
	}

	uuidRegexpMux := validate.UUIDRegexpMux()
	projectRegexpMux := validate.ProjectRegexpMux()

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rUser := r.PathPrefix("/user").Subrouter()
	rUser.Handle("/version", getVersion).Methods("GET")
	rUser.Handle("/job-history", auth.Jwt(authSecret, []string{})(getJobHistory)).
		Methods("GET").HeadersRegexp("Authorization", "^Bearer ")
	rUser.Handle("/balance", auth.Jwt(authSecret, []string{})(getAccountBalance)).
		Methods("GET").HeadersRegexp("Authorization", "^Bearer ")

	jobPathPrefix := fmt.Sprintf("/{uID:%s}/project/{project:%s}/job", uuidRegexpMux, projectRegexpMux)
	rUserAuth := rUser.PathPrefix(jobPathPrefix).HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	rUserAuth.Use(auth.Jwt(authSecret, []string{"user"}))
	rUserAuth.Handle("", auth.UserActive(postJob)).Methods("POST")
	postCancelPath := fmt.Sprintf("/{jID:%s}/cancel", uuidRegexpMux)
	rUserAuth.Handle(postCancelPath,
		auth.JobActive(auth.UserJobMiddleware(postCancelJob))).Methods("POST")

	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			"https://www.emrys.io",
			"http://localhost:8080",
		},
		AllowedHeaders: []string{
			"Origin", "Accept", "Content-Type", "X-Requested-With", "Authorization",
		},
		Debug: debugCors,
	})
	h := c.Handler(r)

	server := http.Server{
		Addr:              ":8080",
		Handler:           log.Log(h),
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
