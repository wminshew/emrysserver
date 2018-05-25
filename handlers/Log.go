// Package handlers handles requests from user and miner clients
package handlers

import (
	"log"
	"net/http"
	// "net/http/httputil"
)

// Log logs request method, URL, & address
func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.Method, r.URL, r.RemoteAddr)
		// TODO: printing RequestDump shows passwords in cleartext on server log...
		// doesn't seem like a best practice
		// Save copy of request for debugging
		// requestDump, err := httputil.DumpRequest(r, true)
		// if err != nil {
		// 	log.Println(err)
		// }
		// log.Println(string(requestDump))
		handler.ServeHTTP(w, r)
	})
}
