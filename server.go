package main

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/handlers"
	"github.com/wminshew/emrysserver/handlers/job"
	"github.com/wminshew/emrysserver/handlers/miner"
	"github.com/wminshew/emrysserver/handlers/user"
	"log"
	"net/http"
)

func main() {
	log.Printf("Initializing database...\n")
	db.Init()
	job.InitCloudStorage()

	log.Printf("Initializing miner pool...\n")
	miner.InitPool()
	go miner.RunPool()

	const httpRedirectPort = ":8080"
	log.Printf("Re-directing port %s...\n", httpRedirectPort)
	go func() {
		log.Fatal(http.ListenAndServe(httpRedirectPort, handlers.Log(http.HandlerFunc(redirect))))
	}()

	const jobProxyPort = ":8081"
	log.Printf("Job proxy server listening on port %s...\n", jobProxyPort)
	go func() {
		rProxy := mux.NewRouter()
		jobR := rProxy.PathPrefix("/job").Subrouter()
		jobR.HandleFunc("/{jID}/bid", job.PostBid).Methods("POST")
		jobR.HandleFunc("/{jID}/auction/success", job.GetAuctionSuccess).Methods("GET")
		jobR.HandleFunc("/{jID}/log", job.PostOutputLog).Methods("POST")
		jobR.HandleFunc("/{jID}/dir", job.PostOutputDir).Methods("POST")
		jobR.HandleFunc("/{jID}/log", job.GetOutputLog).Methods("GET")
		jobR.HandleFunc("/{jID}/dir", job.GetOutputDir).Methods("GET")

		log.Fatal(http.ListenAndServe(jobProxyPort, handlers.Log(rProxy)))
	}()

	r := mux.NewRouter()

	userR := r.PathPrefix("/user").Subrouter()
	userR.HandleFunc("", user.New).Methods("POST")
	userR.HandleFunc("/version", user.GetVersion).Methods("GET")
	userR.HandleFunc("/login", user.Login).Methods("POST")
	userR.HandleFunc("/{uID}/job", user.JWTAuth(user.PostJob)).Methods("POST")
	userR.HandleFunc("/{uID}/job/{jID}/output/log", user.JWTAuth(user.JobAuth(user.GetOutputLog))).Methods("GET")
	userR.HandleFunc("/{uID}/job/{jID}/output/dir", user.JWTAuth(user.JobAuth(user.GetOutputDir))).Methods("GET")

	minerR := r.PathPrefix("/miner").Subrouter()
	minerR.HandleFunc("", miner.New).Methods("POST")
	minerR.HandleFunc("/version", miner.GetVersion).Methods("GET")
	minerR.HandleFunc("/login", miner.Login).Methods("POST")
	minerR.HandleFunc("/{mID}/connect", miner.JWTAuth(miner.Connect)).Methods("GET")
	minerR.HandleFunc("/{mID}/job/{jID}/bid", miner.JWTAuth(miner.PostBid)).Methods("POST")
	minerR.HandleFunc("/{mID}/job/{jID}/image", miner.JWTAuth(miner.JobAuth(miner.Image))).Methods("GET")
	minerR.HandleFunc("/{mID}/job/{jID}/data", miner.JWTAuth(miner.JobAuth(miner.Data))).Methods("GET")
	minerR.HandleFunc("/{mID}/job/{jID}/output/log", miner.JWTAuth(miner.JobAuth(miner.PostOutputLog))).Methods("POST")
	minerR.HandleFunc("/{mID}/job/{jID}/output/dir", miner.JWTAuth(miner.JobAuth(miner.PostOutputDir))).Methods("POST")

	server := http.Server{
		Addr:    ":4430",
		Handler: handlers.Log(r),
	}

	log.Printf("Listening on port %s...\n", server.Addr)
	go log.Fatal(server.ListenAndServeTLS("server.crt", "server.key"))
}

func redirect(w http.ResponseWriter, r *http.Request) {
	newURL := *r.URL
	newURL.Scheme = "https"
	log.Printf("Redirect to: %s", newURL.String())
	http.Redirect(w, r, newURL.String(), http.StatusTemporaryRedirect)
}
