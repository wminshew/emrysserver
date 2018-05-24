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
		jobR.HandleFunc("/{jID}", job.PostOutput).Methods("POST")
		jobR.HandleFunc("/{jID}", job.GetOutput).Methods("GET")

		log.Fatal(http.ListenAndServe(jobProxyPort, handlers.Log(rProxy)))
	}()

	r := mux.NewRouter()

	userR := r.PathPrefix("/user").Subrouter()
	userR.HandleFunc("", user.New).Methods("POST")
	userR.HandleFunc("/login", user.Login).Methods("POST")
	userR.HandleFunc("/job", user.JWTAuth(user.PostJob)).Methods("POST")
	userR.HandleFunc("/job/{jID}/output", user.JWTAuth(user.JobAuth(user.GetOutput))).Methods("GET")

	minerR := r.PathPrefix("/miner").Subrouter()
	minerR.HandleFunc("", miner.New).Methods("POST")
	minerR.HandleFunc("/login", miner.Login).Methods("POST")
	minerR.HandleFunc("/connect", miner.JWTAuth(miner.Connect)).Methods("GET")
	minerR.HandleFunc("/job/{jID}/bid", miner.JWTAuth(miner.Bid)).Methods("POST")
	minerR.HandleFunc("/job/{jID}/image", miner.JWTAuth(miner.JobAuth(miner.Image))).Methods("GET")
	minerR.HandleFunc("/job/{jID}/data", miner.JWTAuth(miner.JobAuth(miner.Data))).Methods("GET")
	minerR.HandleFunc("/job/{jID}/output", miner.JWTAuth(miner.JobAuth(miner.PostOutput))).Methods("POST")

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
