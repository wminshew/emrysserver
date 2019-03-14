package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// getJobCancel tells the miner that the user has canceled the job
var getJobCancel app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	jID := vars["jID"]
	_, err := uuid.FromString(jID)
	if err != nil {
		log.Sugar.Errorw("error parsing job ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	log.Sugar.Infow("watching for job cancellations...",
		"method", r.Method,
		"url", r.URL,
		"jID", jID,
	)

	q := r.URL.Query()
	q.Set("category", fmt.Sprintf("%s-canceled", jID))
	q.Set("timeout", fmt.Sprintf("%d", maxTimeout))
	r.URL.RawQuery = q.Encode()
	jobsManager.SubscriptionHandler(w, r)

	return nil
}
