package user

import (
	// "github.com/wminshew/emrysserver/pkg/flushwriter"
	"github.com/gorilla/mux"
	"github.com/wminshew/check"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
)

// GetOutputLog streams job output to user
func GetOutputLog(w http.ResponseWriter, r *http.Request) {
	// fw := flushwriter.New(w)
	//
	// _, err := fw.Write([]byte("Running image...\n"))
	// if err != nil {
	// 	log.Printf("Error writing to flushWriter: %v\n", err)
	// }

	vars := mux.Vars(r)
	jID := vars["jID"]

	p := path.Join("job", jID)
	u := url.URL{
		Scheme: "http",
		Host:   "localhost:8081",
		Path:   p,
	}
	resp, err := http.Get(u.String())
	if err != nil {
		log.Printf("Error GET %v: %v\n", u.String(), err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	_, _ = io.Copy(w, resp.Body)
	check.Err(resp.Body.Close)
}
