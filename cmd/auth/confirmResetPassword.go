package main

import (
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrys/pkg/validate"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

// confirmResetPassword confirms the account's initial reset-password trigger
var confirmResetPassword app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	tokenStr := r.URL.Query().Get("token")
	claims := &jwt.StandardClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(authSecret), nil
		})
	if err != nil {
		log.Sugar.Infow("error parsing jwt",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing token"}
	}

	if token.Valid {
		log.Sugar.Infow("valid jwt",
			"method", r.Method,
			"url", r.URL,
			"sub", claims.Subject,
		)
	} else {
		log.Sugar.Infow("invalid jwt",
			"method", r.Method,
			"url", r.URL,
			"sub", claims.Subject,
		)
		// TODO: handle expired tokens specially?
		return &app.Error{Code: http.StatusUnauthorized, Message: "invalid token"}
	}

	aUUID, err := uuid.FromString(claims.Subject)
	if err != nil {
		log.Sugar.Errorw("error parsing jwt claims.Subject",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing token claims"}
	}

	c := &creds.Account{}
	if err = json.NewDecoder(r.Body).Decode(c); err != nil {
		log.Sugar.Errorw("error decoding json request body",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json request body"}
	}

	if c.Password == "" {
		log.Sugar.Infow("no password included",
			"method", r.Method,
			"url", r.URL,
			"email", c.Email,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "no password included"}
	} else if !validate.Password(c.Password) {
		log.Sugar.Infow("invalid password",
			"method", r.Method,
			"url", r.URL,
			"email", c.Email,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "invalid password"}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Sugar.Errorw("error hashing password",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if err := db.SetAccountPassword(r, aUUID, string(hashedPassword)); err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
	}
	log.Sugar.Infof("Account %s: password reset", aUUID.String())

	if _, err := w.Write([]byte("Password successfully reset")); err != nil {
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
