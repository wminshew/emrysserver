package main

import (
	"encoding/json"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

const cost = 14

// newMiner creates a new miners entry in database if successful
func newMiner() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
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
		return db.InsertMiner(r, c.Email, string(hashedPassword), mUUID)
	}
}
