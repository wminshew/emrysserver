// package main begins a job server
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
	"github.com/wminshew/emrysserver/pkg/storage"
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
	storage.Init()
	initJobsManager()

	uuidRegexpMux := validate.UUIDRegexpMux()

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	jobPathPrefix := fmt.Sprintf("/job/{jID:%s}", uuidRegexpMux)
	rJob := r.PathPrefix(jobPathPrefix).HeadersRegexp("Authorization", "^Bearer ").Subrouter()

	rJobMiner := rJob.NewRoute().Methods("POST").Subrouter()
	rJobMiner.Use(auth.Jwt(authSecret, []string{"miner"}))
	rJobMiner.Use(auth.MinerJobMiddleware)
	rJobMiner.Use(auth.JobActive)
	rJobMiner.Handle("/log", postOutputLog)
	rJobMiner.Handle("/data", postOutputData)

	rJobUser := rJob.NewRoute().Methods("GET").Subrouter()
	rJobUser.Use(auth.Jwt(authSecret, []string{"user"}))
	rJobUser.Use(auth.UserJobMiddleware)
	rJobUser.Handle("/log", streamOutputLog)
	rJobUser.Handle("/log/download", downloadOutputLog)
	rJobUser.Handle("/data", getOutputData)

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
