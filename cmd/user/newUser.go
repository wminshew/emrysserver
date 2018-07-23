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

// newUser creates a new users entry in database if successful
func newUser() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		c := &creds.User{}
		if err := json.NewDecoder(r.Body).Decode(c); err != nil {
			log.Sugar.Errorw("failed to decode json request body",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json request body"}
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.Password), cost)
		if err != nil {
			log.Sugar.Errorw("failed to hash password",
				"url", r.URL,
				"err", err.Error(),
				"email", c.Email,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		uUUID := uuid.NewV4()
		return db.InsertUser(r, c.Email, string(hashedPassword), uUUID)
	}
}
