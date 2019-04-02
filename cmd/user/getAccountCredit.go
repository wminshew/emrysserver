package main

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// getAccountCredit returns the account's credit
var getAccountCredit app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	aID := r.Header.Get("X-Jwt-Claims-Subject")
	aUUID, err := uuid.FromString(aID)
	if err != nil {
		log.Sugar.Errorw("error parsing account ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	credit, err := db.GetAccountCredit(aUUID)
	if err != nil {
		log.Sugar.Errorw("error getting account credit",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if _, err := w.Write([]byte(fmt.Sprintf("%d", credit))); err != nil {
		log.Sugar.Errorw("error writing account credit",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
