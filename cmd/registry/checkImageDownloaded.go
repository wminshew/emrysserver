package main

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// checkImageDownloaded checks if user has already synced data for this job
func checkImageDownloaded(h http.Handler) http.Handler {
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
		if t, err := db.GetStatusImageDownloaded(r, jUUID); err != nil {
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged err
		} else if !t.IsZero() {
			log.Sugar.Infow("user tried to re-download image",
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
