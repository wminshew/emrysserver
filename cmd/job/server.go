// package main begins the emrys-job server
package main

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"net/http"
)

func main() {
	app.InitLogger()
	defer func() { _ = app.Sugar.Sync() }()
	db.Init()
	defer db.Close()
	initStorage()

	r := mux.NewRouter()
	r.Handle("/healthz", app.Handler(app.HealthCheck)).Methods("GET")
	sr := r.PathPrefix("/job").Subrouter()
	// r.Handle("", app.Handler(newJob)).Methods("POST")
	// r.Handle("/", app.Handler(newJob)).Methods("POST")
	// TODO: proxy bid to job service
	sr.Handle("/{jID}/bid", app.Handler(postBid)).Methods("POST")
	// TODO: proxy auction to job service
	sr.Handle("/{jID}/auction", app.Handler(postAuction)).Methods("POST")
	// TODO: i think i won't need this anymore......
	sr.Handle("/{jID}/auction/success", app.Handler(getAuctionSuccess)).Methods("GET")
	// TODO: proxy data to job service
	// TODO: proxy output to job service
	sr.Handle("/{jID}/log", app.Handler(postOutputLog)).Methods("POST")
	sr.Handle("/{jID}/dir", app.Handler(postOutputDir)).Methods("POST")
	sr.Handle("/{jID}/log", app.Handler(getOutputLog)).Methods("GET")
	sr.Handle("/{jID}/dir", app.Handler(getOutputDir)).Methods("GET")

	server := http.Server{
		Addr:    ":8080",
		Handler: app.Log(r),
	}

	app.Sugar.Infof("Listening on port %s...", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		app.Sugar.Fatalf("Server error: %v", err)
	}
}
