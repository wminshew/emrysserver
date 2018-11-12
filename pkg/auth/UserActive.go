package auth

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// UserActive checks if the user is not suspended
func UserActive(h http.Handler) http.Handler {
	return app.Handler(func(w http.ResponseWriter, r *http.Request) *app.Error {
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

		if suspended, err := db.GetAccountSuspended(r, uUUID); err != nil {
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // err already logged
		} else if suspended {
			log.Sugar.Infow("user is suspended",
				"method", r.Method,
				"url", r.URL,
				"uID", uID,
			)
			return &app.Error{Code: http.StatusUnauthorized, Message: "user is suspended"}
		}

		log.Sugar.Infof("user is active")
		h.ServeHTTP(w, r)
		return nil
	})
}
