package main

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// checkDataSynced checks if user has already synced data for this job
func checkDataSynced(h http.Handler) http.Handler {
	return app.Handler(func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		jID := vars["jID"]
		jUUID, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("error parsing job ID",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}
		if t, err := db.GetStatusDataSynced(r, jUUID); err != nil {
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // err already logged
		} else if !t.IsZero() {
			log.Sugar.Infow("user tried to re-sync data",
				"method", r.Method,
				"url", r.URL,
				"jID", jID,
			)
			return nil
		}
		h.ServeHTTP(w, r)
		return nil
	})
}
