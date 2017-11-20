package main

import (
	"crypto/subtle"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
)

func hello(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "hello, world")
}

func upload(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body from POST /job/upload: %v\n", err)
	}
	log.Printf("Body: %s\n", body)
	io.WriteString(w, "Upload accepted!")
}

var mux map[string]func(http.ResponseWriter, *http.Request)

func main() {
	server := http.Server{
		Addr:    ":8080",
		Handler: Log(Auth(&myHandler{})),
	}

	mux = make(map[string]func(http.ResponseWriter, *http.Request))
	mux["/"] = hello
	mux["/job/upload"] = upload

	log.Printf("Listening on port %s...\n", server.Addr)
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
		log.Printf("%s %s %s\n", r.Method, r.URL, r.RemoteAddr)
		// Save a copy of this request for debugging.
		requestDump, err := httputil.DumpRequest(r, true)
		if err != nil {
			log.Println(err)
		}
		log.Println(string(requestDump))
		handler.ServeHTTP(w, r)
	})
}

func Auth(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || !check(user, pass) {
			realm := "Please provide a username and password."
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized. Please provide username and password, or create an account at https://emrys.io\n"))
			log.Printf("Unauthorized attempt. User: %s\n", user)
			return
		}
		log.Printf("Authorized user: %s\n", user)
		handler.ServeHTTP(w, r)
	})
}

func check(user, pass string) bool {
	username := "admin"
	password := "123456"
	return subtle.ConstantTimeCompare([]byte(user), []byte(username)) == 1 && subtle.ConstantTimeCompare([]byte(pass), []byte(password)) == 1
}
