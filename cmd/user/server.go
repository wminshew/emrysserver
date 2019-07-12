// package main begins a user server
package main

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/account"
	"github.com/stripe/stripe-go/customer"
	"github.com/stripe/stripe-go/invoiceitem"
	"github.com/stripe/stripe-go/loginlink"
	"github.com/stripe/stripe-go/sub"
	"github.com/stripe/stripe-go/transfer"
	"github.com/wminshew/emrys/pkg/validate"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/auth"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	sheets "google.golang.org/api/sheets/v4"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	authSecret                 = os.Getenv("AUTH_SECRET")
	stripeSecretKey            = os.Getenv("STRIPE_SECRET_KEY")
	stripePubKey               = os.Getenv("STRIPE_PUB_KEY")
	stripeWebhookSecretAccount = os.Getenv("STRIPE_WEBHOOK_SECRET_ACCOUNT")
	stripeWebhookSecretConnect = os.Getenv("STRIPE_WEBHOOK_SECRET_CONNECT")
	stripePlanID               = os.Getenv("STRIPE_USER_PLAN_ID")
	stripeInvoiceItemC         *invoiceitem.Client
	stripeTransferC            *transfer.Client
	stripeLoginLinkC           *loginlink.Client
	stripeAccountC             *account.Client
	stripeSubC                 *sub.Client
	stripeCustomerC            *customer.Client
	debugCors                  = (os.Getenv("DEBUG_CORS") == "true")
	debugLog                   = (os.Getenv("DEBUG_LOG") == "true")
	sheetsService              *sheets.Service
)

func main() {
	var err error
	ctx := context.Background()
	log.Init(debugLog, true)
	defer func() {
		if err := log.Sugar.Sync(); err != nil {
			log.Sugar.Errorf("Error syncing log: %v\n", err)
		}
	}()
	if sheetsService, err = sheets.NewService(ctx); err != nil {
		log.Sugar.Errorf("Error initializing google sheets service: %v", err)
		return
	}
	db.Init()
	defer db.Close()

	// TODO: should move all of this into the payments pkg, or a new package which covers all of stripe
	stripeConfig := &stripe.BackendConfig{
		// MaxNetworkRetries: maxRetries, TODO
		LeveledLogger: log.Sugar,
	}
	stripeCustomerC = &customer.Client{
		B:   stripe.GetBackendWithConfig(stripe.APIBackend, stripeConfig),
		Key: stripeSecretKey,
	}
	stripeSubC = &sub.Client{
		B:   stripe.GetBackendWithConfig(stripe.APIBackend, stripeConfig),
		Key: stripeSecretKey,
	}
	stripeAccountC = &account.Client{
		B:   stripe.GetBackendWithConfig(stripe.APIBackend, stripeConfig),
		Key: stripeSecretKey,
	}
	stripeInvoiceItemC = &invoiceitem.Client{
		B:   stripe.GetBackendWithConfig(stripe.APIBackend, stripeConfig),
		Key: stripeSecretKey,
	}
	stripeTransferC = &transfer.Client{
		B:   stripe.GetBackendWithConfig(stripe.APIBackend, stripeConfig),
		Key: stripeSecretKey,
	}
	stripeLoginLinkC = &loginlink.Client{
		B:   stripe.GetBackendWithConfig(stripe.APIBackend, stripeConfig),
		Key: stripeSecretKey,
	}

	uuidRegexpMux := validate.UUIDRegexpMux()
	projectRegexpMux := validate.ProjectRegexpMux()

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods(http.MethodGet)

	rStripe := r.PathPrefix("/stripe").Subrouter()
	rStripe.Handle("/webhook/account", postStripeWebhookAccount).Methods(http.MethodPost)
	rStripe.Handle("/webhook/connect", postStripeWebhookConnect).Methods(http.MethodPost)

	rUser := r.PathPrefix("/user").Subrouter()
	rUser.Handle("/version", getVersion).Methods(http.MethodGet)
	rUser.Handle("/job-history", auth.Jwt(authSecret, []string{})(getJobHistory)).Methods(http.MethodGet)
	rUser.Handle("/credit", auth.Jwt(authSecret, []string{})(getAccountCredit)).Methods(http.MethodGet)
	rUser.Handle("/email", auth.Jwt(authSecret, []string{})(getAccountEmail)).Methods(http.MethodGet)
	rUser.Handle("/feedback", auth.Jwt(authSecret, []string{})(postFeedback)).Methods(http.MethodPost)

	rUser.Handle("/stripe-id", auth.Jwt(authSecret, []string{})(getStripeAccountID)).Methods(http.MethodGet)
	rUser.Handle("/confirm-stripe", auth.Jwt(authSecret, []string{})(postStripeConfirmAccount)).Methods(http.MethodPost)
	rUser.Handle("/stripe/dashboard", auth.Jwt(authSecret, []string{})(getStripeDashboard)).Methods(http.MethodGet)
	rUser.Handle("/stripe/token", auth.Jwt(authSecret, []string{})(postStripeCustomerToken)).Methods(http.MethodPost)
	rUser.Handle("/stripe/last4", auth.Jwt(authSecret, []string{})(getStripeCustomerLast4)).Methods(http.MethodGet)

	jobPathPrefix := fmt.Sprintf("/project/{project:%s}/job", projectRegexpMux)
	rUserAuth := rUser.PathPrefix(jobPathPrefix).Subrouter()
	rUserAuth.Use(auth.Jwt(authSecret, []string{"user"}))
	rUserAuth.Handle("", auth.UserActive(postJob)).Methods(http.MethodPost)
	postCancelPath := fmt.Sprintf("/{jID:%s}/cancel", uuidRegexpMux)
	rUserAuth.Handle(postCancelPath,
		auth.JobActive(auth.UserJobMiddleware(postCancelJob))).Methods(http.MethodPost)

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

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop
	if err := server.Shutdown(ctx); err != nil {
		log.Sugar.Errorf("shutting server down: %v", err)
	}
}
