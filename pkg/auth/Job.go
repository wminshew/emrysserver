package auth

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// UserJobMiddleware applies userJob as router middleware
func UserJobMiddleware() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return userJob(h)
	}
}

// MinerJobMiddleware applies userJob as router middleware
func MinerJobMiddleware() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return minerJob(h)
	}
}

// userJob authorizes an authenticated user for the requested job
func userJob(h http.Handler) app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		jID := vars["jID"]
		jUUID, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("failed to parse job ID",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}
		uID := r.Header.Get("X-Jwt-Claims-Subject")
		uUUID, err := uuid.FromString(uID)
		if err != nil {
			log.Sugar.Errorw("failed to parse user ID from valid jwt",
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		dbuUUID, err := db.GetJobOwner(r, jUUID)
		if err != nil {
			log.Sugar.Errorw("failed to get job owner",
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		if !uuid.Equal(uUUID, dbuUUID) {
			log.Sugar.Errorf("User %v: jwt claims subject does not own job %v (winner: %v)", uUUID, jUUID.String(), dbuUUID)
			return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized jwt"}
		}

		h.ServeHTTP(w, r)
		return nil
	}
}

// minerJob authorizes an authenticated miner for the requested job
func minerJob(h http.Handler) app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		jID := vars["jID"]
		jUUID, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("failed to parse job ID",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}
		mID := r.Header.Get("X-Jwt-Claims-Subject")
		mUUID, err := uuid.FromString(mID)
		if err != nil {
			log.Sugar.Errorw("failed to parse miner ID from valid jwt",
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		dbmUUID, err := db.GetJobWinner(r, jUUID)
		if err != nil {
			log.Sugar.Errorw("failed to get job winner",
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		if !uuid.Equal(mUUID, dbmUUID) {
			log.Sugar.Errorf("Miner %v did not win job %v (winner: %v)", mUUID, jUUID.String(), dbmUUID)
			return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized jwt"}
		}

		h.ServeHTTP(w, r)
		return nil
	}
}
