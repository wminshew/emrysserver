package miner

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/wminshew/emrys/pkg/check"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"time"
)

// PostBid accepts a job.Bid from miner and calls handlers/job.PostBid
func PostBid(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	mID := vars["mID"]
	jID := vars["jID"]
	p := path.Join("job", jID, "bid")
	q := url.Values{
		"mID": []string{mID},
	}
	u := url.URL{
		Scheme:   "http",
		Host:     "localhost:8081",
		Path:     p,
		RawQuery: q.Encode(),
	}
	resp, err := http.Post(u.String(), "application/json", r.Body)
	if err != nil {
		log.Printf("Error POST %v: %v\n", u.String(), err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer check.Err(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		log.Printf("Internal error: Response header error: %v\n", resp.Status)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	winner := resp.Header.Get("Winner")
	if winner != "" {
		t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"exp": time.Now().Add(time.Hour * 1).Unix(),
			"iss": "bid.service",
			"iat": time.Now().Unix(),
			"sub": jID,
		})

		tString, err := t.SignedString([]byte(secret))
		if err != nil {
			log.Printf("Error signing token string: %v\n", err)
			http.Error(w, "Internal error.", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Set-Job-Authorization", tString)
	}

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Internal error: Response header error: %v\n", resp.Status)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
}
