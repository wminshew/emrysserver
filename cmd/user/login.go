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

// login takes user credentials from the request and, if valid, returns a token
var login app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	c := &creds.User{}
	if err := json.NewDecoder(r.Body).Decode(c); err != nil {
		log.Sugar.Errorw("error decoding json request body",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json request body"}
	}

	uUUID, hashedPassword, confirmed, suspended, err := db.GetUserUUIDAndPassword(r, c.Email)
	if err != nil {
		if err == db.ErrUnauthorizedUser {
			return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized user"}
		}
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	} else if !confirmed {
		return &app.Error{Code: http.StatusUnauthorized, Message: "you must confirm your email address before your account is active"}
	} else if suspended {
		return &app.Error{Code: http.StatusUnauthorized, Message: "your account is currently suspended"}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(c.Password)); err != nil {
		log.Sugar.Infow("unauthorized user",
			"method", r.Method,
			"url", r.URL,
			"uID", uUUID,
		)
		return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized user"}
	}

	days := stdDuration
	if d, err := strconv.Atoi(c.Duration); err == nil {
		days = d
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": "emrys.io",
		"exp": time.Now().Add(time.Hour * 24 * time.Duration(days)).Unix(),
		"iss": "emrys.io",
		"iat": time.Now().Unix(),
		"sub": uUUID,
	})

	tokenString, err := token.SignedString([]byte(userSecret))
	if err != nil {
		log.Sugar.Errorw("error signing token",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"uID", uUUID,
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
			"uID", uUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	log.Sugar.Infow("user login",
		"method", r.Method,
		"url", r.URL,
		"sub", uUUID,
	)
	return nil
}
