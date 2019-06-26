// package main begins a job server
package main

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/invoiceitem"
	"github.com/stripe/stripe-go/transfer"
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
	authSecret         = os.Getenv("AUTH_SECRET")
	debugCors          = (os.Getenv("DEBUG_CORS") == "true")
	debugLog           = (os.Getenv("DEBUG_LOG") == "true")
	stripeSecretKey    = os.Getenv("STRIPE_SECRET_KEY")
	stripeInvoiceItemC *invoiceitem.Client
	stripeTransferC    *transfer.Client
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
	initJobsManager()

	stripeConfig := &stripe.BackendConfig{
		// MaxNetworkRetries: maxRetries, TODO
		LeveledLogger: log.Sugar,
	}
	stripeInvoiceItemC = &invoiceitem.Client{
		B:   stripe.GetBackendWithConfig(stripe.APIBackend, stripeConfig),
		Key: stripeSecretKey,
	}
	stripeTransferC = &transfer.Client{
		B:   stripe.GetBackendWithConfig(stripe.APIBackend, stripeConfig),
		Key: stripeSecretKey,
	}

	uuidRegexpMux := validate.UUIDRegexpMux()

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	jobPathPrefix := fmt.Sprintf("/job/{jID:%s}", uuidRegexpMux)
	rJob := r.PathPrefix(jobPathPrefix).HeadersRegexp("Authorization", "^Bearer ").Subrouter()

	rJobMiner := rJob.NewRoute().Subrouter()
	rJobMiner.Use(auth.Jwt(authSecret, []string{"miner"}))
	rJobMiner.Use(auth.MinerJobMiddleware)
	rJobMiner.Use(auth.JobActive)
	rJobMiner.Handle("/log", postOutputLog).Methods("POST")
	rJobMiner.Handle("/data", postOutputData).Methods("POST")
	rJobMiner.Handle("/cancel", getJobCancel).Methods("GET")

	rJobUser := rJob.NewRoute().Subrouter()
	rJobUser.Use(auth.Jwt(authSecret, []string{"user"}))
	rJobUser.Use(auth.UserJobMiddleware)
	rJobUser.Handle("/log", auth.JobActive(streamOutputLog)).Methods("GET")
	rJobUser.Handle("/log/download", downloadOutputLog).Methods("GET")
	rJobUser.Handle("/data", getOutputData).Methods("GET")
	rJobUser.Handle("/data/posted", getJobOutputDataPosted).Methods("GET") // TODO: add JobActive mdlwre?
	rJobUser.Handle("/cancel", postJobCancel).Methods("POST")

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
