package main

import (
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// connect handles miner requests to /miner/connect, establishing
// a pubsub pattern for new jobs to be distributed for bidding
func connect() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		mID := r.Header.Get("X-Jwt-Claims-Subject")
		log.Sugar.Infof("Miner %s connected!", mID)

		q := r.URL.Query()
		q.Set("category", "jobs")
		r.URL.RawQuery = q.Encode()
		jobsManager.SubscriptionHandler(w, r)
		return nil
	}
}
