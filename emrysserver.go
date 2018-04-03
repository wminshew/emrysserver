package main

import (
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/handlers"
	"log"
	"net/http"
)

func main() {
	server := http.Server{}

	mux := http.NewServeMux()
	mux.HandleFunc("/signup/user", handlers.SignUpUser)
	mux.HandleFunc("/signin/user", handlers.SignInUser)

	mux.HandleFunc("/job/upload", handlers.JobUpload)

	server.Addr = ":8080"
	// server.Handler = handlers.Log(handlers.AuthUser(mux))
	server.Handler = handlers.Log(mux)

	log.Printf("Initializing database...\n")
	db.Init()

	log.Printf("Listening on port %s...\n", server.Addr)
	log.Fatal(server.ListenAndServe())
}
