// package main begins a miner server
package main

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/account"
	"github.com/stripe/stripe-go/charge"
	"github.com/wminshew/emrys/pkg/validate"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/auth"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"github.com/wminshew/emrysserver/pkg/storage"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var (
	authSecret      = os.Getenv("AUTH_SECRET")
	sendgridSecret  = os.Getenv("SENDGRID_SECRET")
	stripeSecretKey = os.Getenv("STRIPE_SECRET_KEY")
	debugCors       = (os.Getenv("DEBUG_CORS") == "true")
	debugLog        = (os.Getenv("DEBUG_LOG") == "true")
	minerTimeoutStr = os.Getenv("MINER_TIMEOUT")
	minerTimeout    int
	stripeAccountC  *account.Client
	stripeChargeC   *charge.Client
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
	var err error
	if minerTimeout, err = strconv.Atoi(minerTimeoutStr); err != nil {
		panic(err)
	}
	initMinerManager()

	stripeConfig := &stripe.BackendConfig{
		// MaxNetworkRetries: maxRetries, TODO
		LeveledLogger: log.Sugar,
	}
	stripeAccountC = &account.Client{
		B:   stripe.GetBackendWithConfig(stripe.APIBackend, stripeConfig),
		Key: stripeSecretKey,
	}
	stripeChargeC = &charge.Client{
		B:   stripe.GetBackendWithConfig(stripe.APIBackend, stripeConfig),
		Key: stripeSecretKey,
	}

	uuidRegexpMux := validate.UUIDRegexpMux()

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rMiner := r.PathPrefix("/miner").Subrouter()
	rMiner.Handle("/version", getVersion).Methods("GET")

	rMinerAuth := rMiner.NewRoute().Subrouter()
	rMinerAuth.Use(auth.Jwt(authSecret, []string{"miner"}))
	rMinerAuth.Handle("/connect", auth.MinerActive(connect)).Methods("GET")
	rMinerAuth.Handle("/stats", postMinerStats).Methods("POST")
	postBidPath := fmt.Sprintf("/job/{jID:%s}/bid", uuidRegexpMux)
	rMinerAuth.Handle(postBidPath, auth.JobActive(postBid)).Methods("POST")

	rAuction := r.PathPrefix("/auction").Subrouter()
	rAuction.Use(auth.Jwt(authSecret, []string{"user"}))
	rAuction.Use(auth.UserJobMiddleware)
	rAuction.Use(auth.JobActive)
	postAuctionPath := fmt.Sprintf("/{jID:%s}", uuidRegexpMux)
	rAuction.Handle(postAuctionPath, postAuction).Methods("POST")

	corsR := cors.New(cors.Options{
		AllowedOrigins: []string{
			"https://www.emrys.io",
			"http://localhost:8080",
		},
		AllowedHeaders: []string{
			"Origin", "Accept", "Content-Type", "X-Requested-With", "Authorization",
		},
		Debug: debugCors,
	})
	h := corsR.Handler(r)

	server := http.Server{
		Addr:              ":8080",
		Handler:           log.Log(h),
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       620 * time.Second, // per https://cloud.google.com/load-balancing/docs/https/#timeouts_and_retries
	}

	go func() {
		log.Sugar.Infof("Miner listening on port %s...", server.Addr)
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
