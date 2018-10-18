package payments

import (
	"fmt"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// AuthorizeUser checks the user is authorized to create new jobs
func AuthorizeUser(h http.Handler) http.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
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

		if authorized, err := db.GetUserPaymentAuthorization(r, uUUID); err != nil {
			return &app.Error{Code: http.StatusInternalError, Message: "internal error"} // err already logged
		} else if authorized {
			log.Sugar.Infof("user authorized for payments")
			h.ServeHTTP(w, r)
		} else {
			log.Sugar.Infow("user unauthorized for payments",
				"method", r.Method,
				"url", r.URL,
				"uID", uID,
			)
			return &app.Error{Code: http.StatusPaymentRequired, Message: "no bids received, please try again"}
		}
		return nil
	}
}
