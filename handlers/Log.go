// package handlers
package handlers

import (
	"log"
	"net/http"
	// "net/http/httputil"
)

func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.Method, r.URL, r.RemoteAddr)
		// TODO: printing RequestDump shows passwords in cleartext on server log...
		// doesn't seem like a best practice
		// Save a copy of this request for debugging.
		// requestDump, err := httputil.DumpRequest(r, true)
		// if err != nil {
		// 	log.Println(err)
		// }
		// log.Println(string(requestDump))
		handler.ServeHTTP(w, r)
	})
}
