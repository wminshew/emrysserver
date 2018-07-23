package main

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// postJob handles new jobs posted by users
func postJob() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		uID := vars["uID"]
		uUUID, err := uuid.FromString(uID)
		if err != nil {
			log.Sugar.Errorw("failed to parse user ID",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing user ID"}
		}

		jobID := uuid.NewV4()
		j := &job.Job{
			ID:     jobID,
			UserID: uUUID,
		}
		w.Header().Set("X-Job-ID", j.ID.String())

		return db.InsertJob(r, j.UserID, j.ID)
	}
}
