package main

import (
	"encoding/json"
	"github.com/lib/pq"
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
func newUser(w http.ResponseWriter, r *http.Request) *app.Error {
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
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json request body"}
	}

	u := uuid.NewV4()
	sqlStmt := `
	INSERT INTO users (user_email, password, user_uuid)
	VALUES ($1, $2, $3)
	`
	if _, err = db.Db.Exec(sqlStmt, c.Email, string(hashedPassword), u); err != nil {
		pqErr := err.(*pq.Error)
		if pqErr.Fatal() {
			log.Sugar.Fatalw("failed to insert user",
				"url", r.URL,
				"err", err.Error(),
				"email", c.Email,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw("failed to insert user",
				"url", r.URL,
				"err", err.Error(),
				"email", c.Email,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		}
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	log.Sugar.Infof("User %s successfully added!", c.Email)
	return nil
}
