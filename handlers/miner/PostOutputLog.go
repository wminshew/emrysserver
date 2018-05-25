package miner

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/check"
	"log"
	"net/http"
	"net/url"
	"path"
)

// PostOutputLog receives job output from miner
func PostOutputLog(w http.ResponseWriter, r *http.Request) {
	log.Printf("miner.PostOutputLog!\n")

	// scanner := bufio.NewScanner(r.Body)
	// for scanner.Scan() {
	// 	log.Println(scanner.Text())
	// }

	vars := mux.Vars(r)
	jID := vars["jID"]

	p := path.Join("job", jID)
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
		log.Printf("Response header error: %v\n", resp.Status)
		check.Err(resp.Body.Close)
		return
	}

	// _, _ = io.Copy(w, resp.Body)
	check.Err(resp.Body.Close)
}
