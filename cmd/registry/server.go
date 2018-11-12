// package main begins an registry server
package main

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	// "github.com/rs/cors"
	"github.com/wminshew/emrys/pkg/validate"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/auth"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	authSecret   = os.Getenv("AUTH_SECRET")
	registryHost = os.Getenv("REGISTRY_HOST")
	// debugCors    = (os.Getenv("DEBUG_CORS") == "true")
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

	uuidRegexpMux := validate.UUIDRegexpMux()

	registryURL := url.URL{
		Scheme: "http",
		Host:   registryHost,
	}
	registryRP := httputil.NewSingleHostReverseProxy(&registryURL)
	registryRP.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 640 * time.Second, // server to load balancer timeout + buffer
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rRegistry := r.PathPrefix("/v2").Methods("GET", "HEAD").Subrouter()
	rRegistry.Use(auth.Jwt(authSecret, []string{"miner"}))
	rRegistry.Handle("/", registryRP)

	rBase := rRegistry.PathPrefix("/emrys/base").Subrouter()
	rBase.NewRoute().Handler(registryRP)

	minerJobPrefix := fmt.Sprintf("/miner/{jID:%s}", uuidRegexpMux)
	rJob := rRegistry.PathPrefix(minerJobPrefix).Subrouter()
	rJob.Use(auth.MinerJobMiddleware)
	rJob.Use(auth.JobActive)
	rJob.Use(checkImageDownloaded)
	rJob.NewRoute().Handler(registryRP)

	// c := cors.New(cors.Options{
	// 	AllowedOrigins: []string{
	// 		"https://www.emrys.io",
	// 		"http://localhost:8080",
	// 	},
	// 	AllowedHeaders: []string{
	// 		"Origin", "Accept", "Content-Type", "X-Requested-With", "Authorization",
	// 	},
	// 	Debug: debugCors,
	// })
	// h := c.Handler(r)

	server := http.Server{
		Addr:    ":5000",
		Handler: log.Log(r),
		// Handler:           log.Log(h),
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
