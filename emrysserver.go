package main

import (
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/handlers"
	"github.com/wminshew/emrysserver/handlers/user"
	"log"
	"net/http"
)

func main() {
	server := http.Server{}

	mux := http.NewServeMux()
	mux.HandleFunc("/user/signup", user.SignUp)
	mux.HandleFunc("/user/signin", user.SignIn)

	// mux.HandleFunc("/job/upload", handlers.JobUpload)
	mux.HandleFunc("/job/upload", user.JWTAuth(handlers.JobUpload))

	server.Addr = ":4430"
	// server.Handler = handlers.Log(handlers.AuthUser(mux))
	server.Handler = handlers.Log(mux)

	log.Printf("Initializing database...\n")
	db.Init()

	const httpPort = ":8080"
	log.Printf("Starting http re-direct on port %s...\n", httpPort)
	go func() {
		log.Fatal(http.ListenAndServe(httpPort, http.HandlerFunc(redirect)))
	}()

	log.Printf("Listening on port %s...\n", server.Addr)
	log.Fatal(server.ListenAndServeTLS("server.crt", "server.key"))
}

func redirect(w http.ResponseWriter, r *http.Request) {
	newURL := *r.URL
	newURL.Scheme = "https"
	log.Printf("Redirect to: %s", newURL.String())
	http.Redirect(w, r, newURL.String(), http.StatusTemporaryRedirect)
}
