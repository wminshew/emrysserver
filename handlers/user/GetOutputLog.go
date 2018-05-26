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

// GetOutputLog streams job output to user
func GetOutputLog(w http.ResponseWriter, r *http.Request) {
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

	if resp.StatusCode != http.StatusOK {
		log.Printf("Internal error: Response header error: %v\n", resp.Status)
		check.Err(resp.Body.Close)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	// tee := io.TeeReader(resp.Body, os.Stdout)
	fw := flushwriter.New(w)
	_, _ = io.Copy(fw, resp.Body)
	check.Err(resp.Body.Close)
}
