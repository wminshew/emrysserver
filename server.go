package main

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/handlers"
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
		for {
			log.Fatal(http.ListenAndServe(httpRedirectPort, http.HandlerFunc(redirect)))
		}
	}()

	r := mux.NewRouter()

	userR := r.PathPrefix("/user").Subrouter()
	userR.HandleFunc("", user.New).Methods("POST")
	// userR.HandleFunc("/", user.New).Methods("POST")
	userR.HandleFunc("/login", user.Login).Methods("POST")
	userR.HandleFunc("/job", user.JWTAuth(user.JobUpload)).Methods("POST")

	minerR := r.PathPrefix("/miner").Subrouter()
	minerR.HandleFunc("", miner.New).Methods("POST")
	// minerR.HandleFunc("/", miner.New).Methods("POST")
	minerR.HandleFunc("/login", miner.Login).Methods("POST")
	minerR.HandleFunc("/connect", miner.JWTAuth(miner.Connect)).Methods("GET")

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
