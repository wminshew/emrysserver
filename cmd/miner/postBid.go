package main

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/wminshew/emrysserver/pkg/app"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
)

// postBid accepts a job.Bid from miner and calls handlers/job.PostBid
func postBid() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
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
		m := "POST"
		req, err := http.NewRequest(m, u.String(), r.Body)
		if err != nil {
			app.Sugar.Errorw("failed to create request",
				"url", r.URL,
				"method", m,
				"path", u.String(),
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		req = req.WithContext(r.Context())
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			app.Sugar.Errorw("failed to execute request",
				"url", r.URL,
				"method", m,
				"path", u.String(),
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		defer app.CheckErr(r, resp.Body.Close)

		if resp.StatusCode != http.StatusOK {
			b, _ := ioutil.ReadAll(resp.Body)
			return &app.Error{Code: resp.StatusCode, Message: string(b)}
		}

		winner := resp.Header.Get("Winner")
		if winner != "" {
			t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"exp": time.Now().Add(time.Hour * 1).Unix(),
				"iss": "bid.service",
				"iat": time.Now().Unix(),
				"sub": jID,
			})

			tString, err := t.SignedString([]byte(minerSecret))
			if err != nil {
				app.Sugar.Errorw("failed to sign token",
					"url", r.URL,
					"err", err.Error(),
				)
				return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
			}

			w.Header().Set("Set-Job-Authorization", tString)
		}

		if _, err = io.Copy(w, resp.Body); err != nil {
			app.Sugar.Errorw("failed to copy response body to response writer",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		return nil
	}
}
