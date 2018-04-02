// server handlers
package handlers

import (
	"log"
	"net/http"
)

// UserAuth authenticates users against database
func AuthUser(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || !auth(user, pass) {
			realm := "Please provide a valid username and password."
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized. Please provide valid username and password. Accounts are created at https://emrys.io\n"))
			log.Printf("Unauthorized attempt. User: %s\n", user)
			return
		}
		log.Printf("Authorized user: %s\n", user)
		handler.ServeHTTP(w, r)
	})
}
