package auth

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// JobActive is router middleware to check if the job is active
func JobActive(h http.Handler) http.Handler {
	return app.Handler(func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		jID := vars["jID"]
		jUUID, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("parsing job ID",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}

		active, err := db.GetJobActive(r, jUUID)
		if err != nil {
			log.Sugar.Errorw("checking if job is active",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		if !active {
			log.Sugar.Infow("inactive job",
				"method", r.Method,
				"url", r.URL,
				"jID", jUUID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "inactive job"}
		}

		h.ServeHTTP(w, r)
		return nil
	})
}

// UserJobMiddleware authorizes an authenticated user for the requested job
func UserJobMiddleware(h http.Handler) http.Handler {
	return app.Handler(func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		jID := vars["jID"]
		jUUID, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("parsing job ID",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}
		uID := r.Header.Get("X-Jwt-Claims-Subject")
		uUUID, err := uuid.FromString(uID)
		if err != nil {
			log.Sugar.Errorw("parsing user ID from valid jwt",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		dbuUUID, err := db.GetJobOwner(r, jUUID)
		if err != nil {
			log.Sugar.Errorw("retrieving job owner",
				"method", r.Method,
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

		log.Sugar.Infof("valid user, owns job")
		h.ServeHTTP(w, r)
		return nil
	})
}

// MinerJobMiddleware authorizes an authenticated miner for the requested job
func MinerJobMiddleware(h http.Handler) http.Handler {
	return app.Handler(func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		jID := vars["jID"]
		jUUID, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("parsing job ID",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}
		mID := r.Header.Get("X-Jwt-Claims-Subject")
		mUUID, err := uuid.FromString(mID)
		if err != nil {
			log.Sugar.Errorw("parsing miner ID from valid jwt",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		dbmUUID, err := db.GetJobWinner(r, jUUID)
		if err != nil {
			log.Sugar.Errorw("retrieving job winner",
				"method", r.Method,
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

		log.Sugar.Infof("valid miner, won job")
		h.ServeHTTP(w, r)
		return nil
	})
}
