package main

import (
	"database/sql"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"time"
)

// getPromo returns whether a promo code is valid
var getPromo app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	promo := r.URL.Query().Get("promo")
	if promo == "" {
		return &app.Error{Code: http.StatusBadRequest, Message: "no promo code"}
	}

	promoCredit, expiration, uses, maxUses, err := db.GetPromoInfo(promo)
	if err == sql.ErrNoRows {
		log.Sugar.Infof("promo doesn't exist")
		return &app.Error{Code: http.StatusPaymentRequired, Message: "invalid promo"}
	} else if err != nil {
		log.Sugar.Errorw("error getting promo info",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"promo", promo,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if promoCredit == 0 {
		// should never happen
		log.Sugar.Errorw("promo has no credit set",
			"method", r.Method,
			"url", r.URL,
			"promo", promo,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	} else if expiration.IsZero() {
		// should never happen
		log.Sugar.Errorw("promo has no expiration date set",
			"method", r.Method,
			"url", r.URL,
			"promo", promo,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	} else if expiration.Before(time.Now()) {
		log.Sugar.Infof("promo %s expired as of %s", promo, expiration.Format("2009-01-01"))
		return &app.Error{Code: http.StatusBadRequest, Message: "promo code is expired"}
	} else if uses >= maxUses {
		log.Sugar.Infof("promo %s used max number of times: %d / %d", promo, uses, maxUses)
		return &app.Error{Code: http.StatusBadRequest, Message: "promo code is expired"}
	}

	return nil
}
