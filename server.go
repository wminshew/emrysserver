package main

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/handlers"
	"github.com/wminshew/emrysserver/handlers/job"
	"github.com/wminshew/emrysserver/handlers/miner"
	"github.com/wminshew/emrysserver/handlers/user"
	"github.com/wminshew/emrysserver/pkg/app"
	"net/http"
)

func main() {
	app.InitLogger()
	defer check.Err(app.Sugar.Sync)
	db.Init()
	user.InitStorage()
	job.InitStorage()

	miner.InitPool()
	go miner.RunPool()

	go func() {
		const httpRedirectPort = ":8080"
		app.Sugar.Infof("Re-directing port %s...", httpRedirectPort)
		if err := http.ListenAndServe(httpRedirectPort, handlers.Log(http.HandlerFunc(redirect))); err != nil {
			app.Sugar.Fatalf("Re-directing server error: %v", err)
		}
	}()

	go func() {
		const jobProxyPort = ":8081"
		app.Sugar.Infof("Job proxy server listening on port %s...", jobProxyPort)
		rProxy := mux.NewRouter()
		jobR := rProxy.PathPrefix("/job").Subrouter()
		jobR.Handle("/{jID}/bid", app.Handler(job.PostBid)).Methods("POST")
		jobR.Handle("/{jID}/auction", app.Handler(job.PostAuction)).Methods("POST")
		jobR.Handle("/{jID}/auction/success", app.Handler(job.GetAuctionSuccess)).Methods("GET")
		jobR.Handle("/{jID}/log", app.Handler(job.PostOutputLog)).Methods("POST")
		jobR.Handle("/{jID}/dir", app.Handler(job.PostOutputDir)).Methods("POST")
		jobR.Handle("/{jID}/log", app.Handler(job.GetOutputLog)).Methods("GET")
		jobR.Handle("/{jID}/dir", app.Handler(job.GetOutputDir)).Methods("GET")

		if err := http.ListenAndServe(jobProxyPort, handlers.Log(rProxy)); err != nil {
			app.Sugar.Fatalf("Job proxy server error: %v", err)
		}
	}()

	r := mux.NewRouter()

	userR := r.PathPrefix("/user").Subrouter()
	userR.Handle("", app.Handler(user.New)).Methods("POST")
	userR.Handle("/version", app.Handler(user.GetVersion)).Methods("GET")
	userR.Handle("/login", app.Handler(user.Login)).Methods("POST")
	userR.Handle("/{uID}/job", app.Handler(user.JWTAuth(user.PostJob))).Methods("POST")
	userR.Handle("/{uID}/job/{jID}/image", app.Handler(user.JWTAuth(user.JobAuth(user.BuildImage)))).Methods("POST")
	userR.Handle("/{uID}/job/{jID}/auction", app.Handler(user.JWTAuth(user.JobAuth(user.RunAuction)))).Methods("POST")
	userR.Handle("/{uID}/job/{jID}/output/log", app.Handler(user.JWTAuth(user.JobAuth(user.GetOutputLog)))).Methods("GET")
	userR.Handle("/{uID}/job/{jID}/output/dir", app.Handler(user.JWTAuth(user.JobAuth(user.GetOutputDir)))).Methods("GET")

	minerR := r.PathPrefix("/miner").Subrouter()
	minerR.Handle("", app.Handler(miner.New)).Methods("POST")
	minerR.Handle("/version", app.Handler(miner.GetVersion)).Methods("GET")
	minerR.Handle("/login", app.Handler(miner.Login)).Methods("POST")
	minerR.Handle("/job/{jID}/auction", app.Handler(miner.PostAuction)).Methods("POST")
	minerR.Handle("/{mID}/connect", app.Handler(miner.JWTAuth(miner.Connect))).Methods("GET")
	minerR.Handle("/{mID}/job/{jID}/bid", app.Handler(miner.JWTAuth(miner.PostBid))).Methods("POST")
	minerR.Handle("/{mID}/job/{jID}/image", app.Handler(miner.JWTAuth(miner.JobAuth(miner.Image)))).Methods("GET")
	minerR.Handle("/{mID}/job/{jID}/data", app.Handler(miner.JWTAuth(miner.JobAuth(miner.Data)))).Methods("GET")
	minerR.Handle("/{mID}/job/{jID}/output/log", app.Handler(miner.JWTAuth(miner.JobAuth(miner.PostOutputLog)))).Methods("POST")
	minerR.Handle("/{mID}/job/{jID}/output/dir", app.Handler(miner.JWTAuth(miner.JobAuth(miner.PostOutputDir)))).Methods("POST")

	server := http.Server{
		Addr:    ":4430",
		Handler: handlers.Log(r),
	}

	app.Sugar.Infof("Listening on port %s...", server.Addr)
	if err := server.ListenAndServeTLS("server.crt", "server.key"); err != nil {
		app.Sugar.Fatalf("Server error: %v", err)
	}
}

func redirect(w http.ResponseWriter, r *http.Request) {
	newURL := *r.URL
	newURL.Scheme = "https"
	app.Sugar.Infof("Redirect to: %s", newURL.String())
	http.Redirect(w, r, newURL.String(), http.StatusTemporaryRedirect)
}
