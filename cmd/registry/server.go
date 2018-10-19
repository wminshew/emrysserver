// package main begins an registry server
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
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
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

	registryURL := url.URL{
		Scheme: "http",
		Host:   registryHost,
	}
	registryRP := httputil.NewSingleHostReverseProxy(&registryURL)

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rRegistry := r.PathPrefix("/v2").Methods("GET", "HEAD").Subrouter()
	rRegistry.Use(auth.Jwt(minerSecret))
	rRegistry.Handle("/", registryRP)

	rBase := rRegistry.PathPrefix("/emrys/base").Subrouter()
	rBase.NewRoute().Handler(registryRP)

	uuidRegexpMux := validate.UUIDRegexpMux()
	rJob := rRegistry.PathPrefix(fmt.Sprintf("/miner/{jID:%s}", uuidRegexpMux)).Subrouter()
	rJob.Use(auth.MinerJobMiddleware)
	rJob.Use(auth.JobActive)
	rJob.Use(checkImageDownloaded)
	rJob.NewRoute().Handler(registryRP)

	server := http.Server{
		Addr:              ":5000",
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
