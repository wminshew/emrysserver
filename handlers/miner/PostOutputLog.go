package miner

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/emrys/pkg/check"
	"log"
	"net/http"
	"net/url"
	"path"
)

// PostOutputLog receives job output from miner
func PostOutputLog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jID := vars["jID"]
	p := path.Join("job", jID, "log")
	u := url.URL{
		Scheme: "http",
		Host:   "localhost:8081",
		Path:   p,
	}
	resp, err := http.Post(u.String(), "text/plain", r.Body)
	if err != nil {
		log.Printf("Error POST %v: %v\n", u.String(), err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Internal error: Response header error: %v\n", resp.Status)
		check.Err(resp.Body.Close)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	check.Err(resp.Body.Close)
}
