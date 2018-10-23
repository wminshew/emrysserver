package auth

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// MinerActive checks if the miner is not suspended
func MinerActive(h http.Handler) http.Handler {
	return app.Handler(func(w http.ResponseWriter, r *http.Request) *app.Error {
		mID := r.Header.Get("X-Jwt-Claims-Subject")
		mUUID, err := uuid.FromString(mID)
		if err != nil {
			log.Sugar.Errorw("error parsing miner ID",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing miner ID"}
		}

		if suspended, err := db.GetMinerSuspended(r, mUUID); err != nil {
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // err already logged
		} else if suspended {
			log.Sugar.Infow("miner is suspended",
				"method", r.Method,
				"url", r.URL,
				"mID", mID,
			)
			return &app.Error{Code: http.StatusUnauthorized, Message: "miner is suspended"}
		}

		log.Sugar.Infof("miner is active")
		h.ServeHTTP(w, r)
		return nil
	})
}
