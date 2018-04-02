package main

import (
	"github.com/wminshew/emrysserver/handlers"
	"log"
	"net/http"
)

func main() {
	server := http.Server{}

	mux := http.NewServeMux()
	mux.HandleFunc("/job/upload", handlers.JobUpload)

	server.Addr = ":8080"
	server.Handler = handlers.Log(handlers.AuthUser(mux))

	log.Printf("Listening on port %s...\n", server.Addr)
	log.Fatal(server.ListenAndServe())
}
