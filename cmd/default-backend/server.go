// package main begins a default-backend server
package main

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	debugCors = (os.Getenv("DEBUG_CORS") == "true")
	debugLog  = (os.Getenv("DEBUG_LOG") == "true")
)

func main() {
	log.Init(debugLog, true)
	defer func() {
		if err := log.Sugar.Sync(); err != nil {
			log.Sugar.Errorf("Error syncing log: %v\n", err)
		}
	}()

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods(http.MethodGet)

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
