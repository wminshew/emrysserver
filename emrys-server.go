package main

import (
	"crypto/subtle"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"syscall"
)

func hello(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "hello, world")
}

func upload(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		maxMemory := int64(1) << 31
		err := r.ParseMultipartForm(maxMemory)
		if err != nil {
			log.Printf("Error parsing request: %v\n", err)
		}

		trainTempFile, trainHandler, err := r.FormFile("Train")
		if err != nil {
			log.Printf("Error opening form file: %v\n", err)
		}
		defer trainTempFile.Close()

		username, _, _ := r.BasicAuth()
		trainDir := "./user-upload/" + username + "/"
		// TODO: THIS FEELS DANGEROUS; IS THERE A SAFER WAY?
		// error behavior without adjusting umask:
		// directory without execution / writing bits cannot be written to
		oldUmask := syscall.Umask(022)
		log.Printf("Old Umask: %v\n", oldUmask)
		if err = os.MkdirAll(trainDir, 0777); err != nil {
			log.Printf("Error creating directory %s: %v\n", trainDir, err)
		}
		_ = syscall.Umask(oldUmask)
		trainPath := trainDir + trainHandler.Filename
		f, err := os.OpenFile(trainPath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Printf("Error opening new file for saving: %v\n", err)
		}
		defer f.Close()

		n, err := io.Copy(f, trainTempFile)
		if err != nil {
			log.Printf("Error copying file from POST /job/upload: %v\n", err)
		}

		// send response to client
		io.WriteString(w, fmt.Sprintf("%d bytes recieved and saved.\n", n))
	} else {
		log.Printf("Upload received non-POST method.\n")
		io.WriteString(w, "Upload only receives POSTs.\n")
	}
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
