package user

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrysserver/pkg/flushwriter"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
)

// GetOutputDir streams job output to user
func GetOutputDir(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jID := vars["jID"]
	p := path.Join("job", jID, "dir")
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

	if resp.StatusCode != http.StatusOK {
		log.Printf("Internal error: Response header error: %v\n", resp.Status)
		check.Err(resp.Body.Close)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	fw := flushwriter.New(w)
	_, _ = io.Copy(fw, resp.Body)
	check.Err(resp.Body.Close)
}
