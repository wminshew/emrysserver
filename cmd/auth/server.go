// package main begins an auth server
package main

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var (
	authSecret    = os.Getenv("AUTH_SECRET")
	debugCors     = (os.Getenv("DEBUG_CORS") == "true")
	newUserCredit int
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
	var err error
	if newUserCredit, err = strconv.Atoi(os.Getenv("NEW_USER_CREDIT")); err != nil {
		log.Sugar.Errorf("Error converting string to int: %v", err)
		return
	}

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rAuth := r.PathPrefix("/auth").Subrouter()
	rAuth.Handle("/account", newAccount).Methods("POST")
	rAuth.Handle("/confirm-email", confirmEmail).Methods("POST")
	rAuth.Handle("/reset-password", resetPassword).Methods("POST")
	rAuth.Handle("/confirm-reset-password", confirmResetPassword).Methods("POST")
	rAuth.Handle("/token", refreshToken).Methods("POST").Queries("grant_type", "token")
	rAuth.Handle("/token", login).Methods("POST")

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
		ReadHeaderTimeout: 5 * time.Second,
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
