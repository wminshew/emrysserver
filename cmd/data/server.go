// package main begins a data server
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
	"path"
	"syscall"
	"time"
)

var (
	authSecret = os.Getenv("AUTH_SECRET")
	debugCors  = (os.Getenv("DEBUG_CORS") == "true")
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
	storage.Init()
	initMetadataSync()
	initDiskManager()
	done := make(chan struct{}, 1)
	go func() {
		for {
			if err := checkAndEvictProjects(); err != nil {
				log.Sugar.Errorf("Error managing disk utilization: %v\n", err)
			}
			select {
			case <-done:
				return
			case <-time.After(time.Duration(pvcPeriodSec) * time.Second):
			}
		}
	}()

	uuidRegexpMux := validate.UUIDRegexpMux()
	projectRegexpMux := validate.ProjectRegexpMux()

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(app.APINotFound)
	r.HandleFunc("/healthz", app.HealthCheck).Methods("GET")

	rDataUser := r.PathPrefix("/user").HeadersRegexp("Authorization", "^Bearer ").Subrouter()
	rDataUser.Use(auth.Jwt(authSecret, []string{"user"}))
	rDataUser.Use(auth.UserJobMiddleware)
	rDataUser.Use(auth.JobActive)
	rDataUser.Use(checkDataSynced)
	syncUserPath := fmt.Sprintf("/project/{project:%s}/job/{jID}", projectRegexpMux)
	rDataUser.Handle(syncUserPath, syncUser).Methods("POST")
	uploadDataPath := path.Join(syncUserPath, "{relPath:.*}")
	rDataUser.Handle(uploadDataPath, uploadData).Methods("PUT")

	rDataMiner := r.PathPrefix("/miner").HeadersRegexp("Authorization", "^Bearer ").Methods("GET").Subrouter()
	getDataPath := fmt.Sprintf("/job/{jID:%s}", uuidRegexpMux)
	rDataMiner.Handle(getDataPath, getData)
	rDataMiner.Use(auth.Jwt(authSecret, []string{"miner"}))
	rDataMiner.Use(auth.MinerJobMiddleware)
	rDataMiner.Use(auth.JobActive)

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
	close(done)
	if err := server.Shutdown(ctx); err != nil {
		log.Sugar.Errorf("shutting server down: %v", err)
	}
}
