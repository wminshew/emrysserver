// package main begins an image server
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
	authSecret   = os.Getenv("AUTH_SECRET")
	registryHost = os.Getenv("REGISTRY_HOST")
	debugCors    = (os.Getenv("DEBUG_CORS") == "true")
	debugLog     = (os.Getenv("DEBUG_LOG") == "true")
)

func main() {
	log.Init(debugLog, false)
	defer func() {
		if err := log.Sugar.Sync(); err != nil {
			log.Sugar.Errorf("Error syncing log: %v\n", err)
		}
	}()
	db.Init()
	defer db.Close()
	storage.Init()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	initDocker(ctx)
	defer func() {
		if err := dClient.Close(); err != nil {
			log.Sugar.Errorf("Error closing docker client: %v\n", err)
		}
	}()

	uuidRegexpMux := validate.UUIDRegexpMux()
	projectRegexpMux := validate.ProjectRegexpMux()

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods(http.MethodGet)

	rImage := r.PathPrefix("/image").Methods(http.MethodPost).Subrouter()

	rImageMiner := rImage.PathPrefix("/downloaded").Subrouter()
	rImageMiner.Use(auth.Jwt(authSecret, []string{"miner"}))
	rImageMiner.Use(auth.MinerJobMiddleware)
	rImageMiner.Use(auth.JobActive)
	postImageDownloadedPath := fmt.Sprintf("/{jID:%s}", uuidRegexpMux)
	rImageMiner.Handle(postImageDownloadedPath, imageDownloaded)

	buildImagePathPrefix := fmt.Sprintf("/{project:%s}", projectRegexpMux)
	rImageUser := rImage.PathPrefix(buildImagePathPrefix).Subrouter()
	rImageUser.Use(auth.Jwt(authSecret, []string{"user"}))
	rImageUser.Use(auth.UserJobMiddleware)
	rImageUser.Use(auth.JobActive)
	postBuildImagePath := fmt.Sprintf("/{jID:%s}", uuidRegexpMux)
	rImageUser.Handle(postBuildImagePath, buildImage)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			"https://www.emrys.io",
			"http://localhost:8080",
		},
		AllowedHeaders: []string{
			"*",
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

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop
	if err := server.Shutdown(ctx); err != nil {
		log.Sugar.Errorf("shutting server down: %v", err)
	}
}
