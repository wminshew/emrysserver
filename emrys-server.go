package main

import (
	"crypto/subtle"
	"fmt"
	"io"
	"log"
	"net/http"
)

func hello(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "hello, world")
}

var mux map[string]func(http.ResponseWriter, *http.Request)

func main() {
	server := http.Server{
		Addr:    ":8080",
		Handler: Log(Auth(&myHandler{})),
	}

	mux = make(map[string]func(http.ResponseWriter, *http.Request))
	mux["/"] = hello

	fmt.Printf("Listening on port %s...\n", server.Addr)
	log.Fatal(server.ListenAndServe())
}

type myHandler struct{}

func (*myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := mux[r.URL.String()]; ok {
		h(w, r)
		return
	}

	io.WriteString(w, "My server: "+r.URL.String())
}

func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func Auth(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || !check(user, pass) {
			// http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			realm := "test"
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			w.WriteHeader(401)
			w.Write([]byte("Unauthorized.\n"))
			fmt.Printf("Unauthorized attempt. User: %s\n", user)
			return
		}
		fmt.Printf("Authorized user: %s\n", user)
		handler.ServeHTTP(w, r)
	})
}

func check(user, pass string) bool {
	username := "admin"
	password := "123456"
	// fmt.Printf("user: %s\n", user)
	// fmt.Printf("username: %s\n", username)
	// fmt.Printf("pass: %s\n", pass)
	// fmt.Printf("password: %s\n", password)
	// fmt.Printf("user v username: %v\n", subtle.ConstantTimeCompare([]byte(user), []byte(username)))
	// fmt.Printf("pass v password: %v\n", subtle.ConstantTimeCompare([]byte(pass), []byte(password)))
	// fmt.Printf("authorized? %v\n", subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1)
	return subtle.ConstantTimeCompare([]byte(user), []byte(username)) == 1 && subtle.ConstantTimeCompare([]byte(pass), []byte(password)) == 1
}
