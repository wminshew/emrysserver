package main

import (
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
	pool := miner.NewPool()
	go pool.Run()

	const httpRedirectPort = ":8080"
	log.Printf("Re-directing port %s...\n", httpRedirectPort)
	go func() {
		for {
			log.Fatal(http.ListenAndServe(httpRedirectPort, http.HandlerFunc(redirect)))
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/user/signup", user.SignUp)
	mux.HandleFunc("/user/signin", user.SignIn)
	mux.HandleFunc("/job/upload", user.JWTAuth(handlers.JobUpload))

	mux.HandleFunc("/miner/signup", miner.SignUp)
	mux.HandleFunc("/miner/signin", miner.SignIn)
	mux.HandleFunc("/miner/connect", miner.JWTAuth(miner.Connect(pool)))

	server := http.Server{
		Addr:    ":4430",
		Handler: handlers.Log(mux),
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
