package main

import (
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/email"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"time"
)

// resetPassword begins the reset-password process by emailing the account with a token & link
var resetPassword app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	c := &creds.Account{}
	if err := json.NewDecoder(r.Body).Decode(c); err != nil {
		log.Sugar.Errorw("error decoding json request body",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json request body"}
	}

	aUUID, _, _, _, confirmed, _, err := db.GetAccountUUIDAndPassword(r, c.Email)
	if err != nil {
		// already logged
		if err == db.ErrUnauthorizedAccount {
			return &app.Error{Code: http.StatusBadRequest, Message: "no account with this email exists"}
		}
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	} else if !confirmed {
		log.Sugar.Errorw("unconfirmed account requested password reset",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "account email still pending confirmation"}
	}
	log.Sugar.Infof("Account %s: password reset requested", aUUID.String())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": "emrys.io",
		"exp": time.Now().Add(time.Hour * 24).Unix(),
		"iss": "emrys.io",
		"iat": time.Now().Unix(),
		"sub": aUUID,
	})
	tokenString, err := token.SignedString([]byte(authSecret))
	if err != nil {
		log.Sugar.Errorw("error signing token",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if err := email.SendResetPassword(c.Email, tokenString); err != nil {
		log.Sugar.Errorw("error sending reset password email",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if _, err := w.Write([]byte("Email sent. Please check your inbox and follow the link to confirm.")); err != nil {
		log.Sugar.Errorw("error writing response",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	return nil
}
