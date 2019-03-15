package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// postJobCancel tells the miner that the user has canceled the job
var postJobCancel app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
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

	if err := jobsManager.Publish(fmt.Sprintf("%s-canceled", jID), struct{}{}); err != nil {
		log.Sugar.Errorw("error publishing empty struct for job canceled",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
