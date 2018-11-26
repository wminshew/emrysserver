package main

import (
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
	"time"
)

const (
	stdDuration = 7
)

// login takes account credentials from the request and, if valid, returns a token
var login app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	c := &creds.Account{}
	if err := json.NewDecoder(r.Body).Decode(c); err != nil {
		log.Sugar.Errorw("error decoding json request body",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json request body"}
	}

	aUUID, hashedPassword, isUser, isMiner, confirmed, suspended, beta, err := db.GetAccountUUIDAndPassword(r, c.Email)
	if err != nil {
		if err == db.ErrUnauthorizedAccount {
			return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized account"}
		}
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	} else if !confirmed {
		return &app.Error{Code: http.StatusUnauthorized, Message: "you must confirm your email address before your account is active"}
	} else if !beta {
		return &app.Error{Code: http.StatusUnauthorized, Message: "you haven't been granted beta access yet"}
	} else if suspended {
		return &app.Error{Code: http.StatusUnauthorized, Message: "your account is currently suspended"}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(c.Password)); err != nil {
		log.Sugar.Infow("unauthorized account",
			"method", r.Method,
			"url", r.URL,
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized account"}
	}

	days := stdDuration
	duration := r.URL.Query().Get("duration")
	if d, err := strconv.Atoi(duration); err == nil {
		days = d
	}
	scope := []string{}
	if isUser {
		scope = append(scope, "user")
	}
	if isMiner {
		scope = append(scope, "miner")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud":   "emrys.io",
		"exp":   time.Now().Add(time.Hour * 24 * time.Duration(days)).Unix(),
		"iss":   "emrys.io",
		"iat":   time.Now().Unix(),
		"sub":   aUUID,
		"scope": scope,
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

	resp := creds.LoginResp{
		Token: tokenString,
	}
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		log.Sugar.Errorw("error encoding json response",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	log.Sugar.Infow("account login",
		"method", r.Method,
		"url", r.URL,
		"sub", aUUID,
	)
	return nil
}
