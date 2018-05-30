// Package handlers handles requests from user and miner clients
package handlers

import (
	"log"
	"net/http"
)

// Log logs request method, URL, & address
func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.Method, r.URL, r.RemoteAddr)
		handler.ServeHTTP(w, r)
	})
}
