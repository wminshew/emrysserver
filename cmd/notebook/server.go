// package main begins a notebook server
package main

import (
	"context"
	// "fmt"
	"github.com/gorilla/mux"
	// "github.com/rs/cors"
	// "github.com/wminshew/emrys/pkg/validate"
	"github.com/wminshew/emrysserver/pkg/app"
	// "github.com/wminshew/emrysserver/pkg/auth"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	// "github.com/wminshew/emrysserver/pkg/storage"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//
// var (
// 	authSecret = os.Getenv("AUTH_SECRET")
// 	debugCors  = (os.Getenv("DEBUG_CORS") == "true")
// )

func main() {
	log.Init()
	defer func() {
		if err := log.Sugar.Sync(); err != nil {
			log.Sugar.Errorf("Error syncing log: %v\n", err)
		}
	}()
	db.Init()
	defer db.Close()
	// storage.Init()

	// uuidRegexpMux := validate.UUIDRegexpMux()

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	// notebookPathPrefix := fmt.Sprintf("/notebook/{jID:%s}", uuidRegexpMux)
	// rNotebook := r.PathPrefix(notebookPathPrefix).HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	//
	// rNotebookMiner := rNotebook.NewRoute().Methods("POST").Subrouter()
	// rNotebookMiner.Use(auth.Jwt(authSecret, []string{"miner"}))
	// rNotebookMiner.Use(auth.MinerNotebookMiddleware)
	// rNotebookMiner.Use(auth.NotebookActive)
	// rNotebookMiner.Handle("/log", postOutputLog)
	// rNotebookMiner.Handle("/data", postOutputData)
	//
	// rNotebookUser := rNotebook.NewRoute().Methods("GET").Subrouter()
	// rNotebookUser.Use(auth.Jwt(authSecret, []string{"user"}))
	// rNotebookUser.Use(auth.UserNotebookMiddleware)
	// rNotebookUser.Handle("/log", streamOutputLog)
	// rNotebookUser.Handle("/log/download", downloadOutputLog)
	// rNotebookUser.Handle("/data", getOutputData)

	// c := cors.New(cors.Options{
	// 	AllowedOrigins: []string{
	// 		"https://www.emrys.io",
	// 	},
	// 	AllowedHeaders: []string{
	// 		"Origin", "Accept", "Content-Type", "X-Requested-With", "Authorization",
	// 	},
	// 	Debug: debugCors,
	// })
	// h := c.Handler(r)

	server := http.Server{
		Addr: ":8080",
		// Handler:           log.Log(h),
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
