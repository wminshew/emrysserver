package main

import (
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/email"
	"github.com/wminshew/emrysserver/pkg/log"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

const cost = 14

// newMiner creates a new miners entry in database if successful
var newMiner app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	c := &creds.Miner{}
	if err := json.NewDecoder(r.Body).Decode(c); err != nil {
		log.Sugar.Errorw("error decoding json request body",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json request body"}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.Password), cost)
	if err != nil {
		log.Sugar.Errorw("error hashing password",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"email", c.Email,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	mUUID := uuid.NewV4()
	if err := db.InsertMiner(r, c.Email, string(hashedPassword), mUUID); err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
	}
	log.Sugar.Infof("Miner %s (%s) successfully added!", c.Email, mUUID.String())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": "emrys.io",
		"exp": time.Now().Add(time.Hour * 72).Unix(), // TODO: make shorter
		"iss": "emrys.io",
		"iat": time.Now().Unix(),
		"sub": mUUID,
	})
	tokenString, err := token.SignedString([]byte(minerSecret))
	if err != nil {
		log.Sugar.Errorw("error signing token",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"mID", mUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	client := "miner"
	if err := email.SendEmailConfirmation(client, c.Email, tokenString); err != nil {
		log.Sugar.Errorw("error sending user confirmation email",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"mID", mUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
