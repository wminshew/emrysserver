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

// login takes miner credentials from the request and, if valid, returns a token
func login() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		c := &creds.Miner{}
		if err := json.NewDecoder(r.Body).Decode(c); err != nil {
			log.Sugar.Errorw("error decoding json request body",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json request body"}
		}

		mUUID, hashedPassword, err := db.GetMinerUUIDAndPassword(r, c.Email)
		if err != nil {
			if err == db.ErrUnauthorizedMiner {
				return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized miner"}
			}
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(c.Password)); err != nil {
			log.Sugar.Infow("unauthorized miner",
				"url", r.URL,
				"mID", mUUID,
				"email", c.Email,
			)
			return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized miner"}
		}

		days := stdDuration
		if d, err := strconv.Atoi(c.Duration); err == nil {
			days = d
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"aud":   "emrys.io",
			"exp":   time.Now().Add(time.Hour * 24 * time.Duration(days)).Unix(),
			"iss":   "emrys.io",
			"iat":   time.Now().Unix(),
			"email": c.Email,
			"sub":   mUUID,
		})

		tokenString, err := token.SignedString([]byte(minerSecret))
		if err != nil {
			log.Sugar.Errorw("error signing token",
				"url", r.URL,
				"err", err.Error(),
				"mID", mUUID,
				"email", c.Email,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		resp := creds.LoginResp{
			Token: tokenString,
		}
		if err = json.NewEncoder(w).Encode(resp); err != nil {
			log.Sugar.Errorw("error encoding json response",
				"url", r.URL,
				"err", err.Error(),
				"mID", mUUID,
				"email", c.Email,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		log.Sugar.Infow("miner login",
			"url", r.URL,
			"sub", mUUID,
			"email", c.Email,
		)
		return nil
	}
}
