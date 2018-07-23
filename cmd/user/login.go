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
func login() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		c := &creds.User{}
		if err := json.NewDecoder(r.Body).Decode(c); err != nil {
			log.Sugar.Errorw("failed to decode json request body",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json request body"}
		}

		uUUID, hashedPassword, err := db.GetUserUUIDAndPassword(r, c.Email)
		if err != nil {
			if err == db.ErrUnauthorizedUser {
				return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized user"}
			}
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(c.Password)); err != nil {
			log.Sugar.Infow("unauthorized user",
				"url", r.URL,
				"uID", uUUID,
				"email", c.Email,
			)
			return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized user"}
		}

		days := stdDuration
		if d, err := strconv.Atoi(c.Duration); err == nil {
			days = d
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"exp":   time.Now().Add(time.Hour * 24 * time.Duration(days)).Unix(),
			"iss":   "auth.service",
			"iat":   time.Now().Unix(),
			"email": c.Email,
			"sub":   uUUID,
		})

		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			log.Sugar.Errorw("failed to sign token",
				"url", r.URL,
				"err", err.Error(),
				"uID", uUUID,
				"email", c.Email,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		resp := creds.LoginResp{
			Token: tokenString,
		}
		if err = json.NewEncoder(w).Encode(resp); err != nil {
			log.Sugar.Errorw("failed to encode json response",
				"url", r.URL,
				"err", err.Error(),
				"uID", uUUID,
				"email", c.Email,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		log.Sugar.Infow("user login",
			"url", r.URL,
			"sub", uUUID,
			"email", c.Email,
		)
		return nil
	}
}
