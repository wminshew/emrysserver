package main

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// postJob handles new jobs posted by users
var postJob app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	project := vars["project"]
	uID := r.Header.Get("X-Jwt-Claims-Subject")
	uUUID, err := uuid.FromString(uID)
	if err != nil {
		log.Sugar.Errorw("error parsing user ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing user ID"}
	}

	jobID := uuid.NewV4()
	w.Header().Set("X-Job-ID", jobID.String())

	return db.InsertJob(r, uUUID, project, jobID)
}
