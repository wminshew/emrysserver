// package main begins an image server
package main

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/auth"
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

	registryURL := url.URL{
		Scheme: "http",
		Host:   registryHost,
	}
	registryRP := httputil.NewSingleHostReverseProxy(&registryURL)

	r := mux.NewRouter()
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	// rRegistry := r.PathPrefix("/v2").Subrouter()
	rRegistry := r.PathPrefix("/v2").Methods("GET", "HEAD").Subrouter()
	rRegistry.NewRoute().Handler(registryRP)
	rRegistry.Use(auth.Jwt(minerSecret))
	// r.Handle("/v2", registryRP)
	//
	// rBase := r.PathPrefix("/v2/emrys/base").Methods("GET", "HEAD").Subrouter()
	// rBase.NewRoute().Handler(registryRP)
	// rBase.Use(auth.Jwt(minerSecret))
	//
	// rRegistry := r.PathPrefix("/v2/miner/{jID}").Methods("GET", "HEAD").Subrouter()
	// rRegistry.NewRoute().Handler(registryRP)
	// rRegistry.Use(auth.Jwt(minerSecret))
	// rRegistry.Use(auth.MinerJobMiddleware())
	// rRegistry.Use(auth.JobActive())

	server := http.Server{
		Addr:              ":5000",
		Handler:           log.Log(r),
		ReadHeaderTimeout: 5 * time.Second,
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
