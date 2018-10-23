package main

import (
	"fmt"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// connect handles miner requests to /miner/connect, establishing
// a pubsub pattern for new jobs to be distributed for bidding
var connect app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	mID := r.Header.Get("X-Jwt-Claims-Subject")
	log.Sugar.Infof("Miner %s connected!", mID)

	q := r.URL.Query()
	q.Set("category", "jobs")
	q.Set("timeout", fmt.Sprintf("%d", maxTimeout))
	r.URL.RawQuery = q.Encode()
	jobsManager.SubscriptionHandler(w, r)
	return nil
}
