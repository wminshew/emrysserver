package miner

import (
	"encoding/json"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/pkg/app"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

const cost = 14

// New creates a new miners entry in database if successful
func New(w http.ResponseWriter, r *http.Request) *app.Error {
	c := &creds.Miner{}
	err := json.NewDecoder(r.Body).Decode(c)
	if err != nil {
		app.Sugar.Errorw("failed to decode json request body",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json request body"}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.Password), cost)

	u := uuid.NewV4()
	sqlStmt := `
	INSERT INTO miners (miner_email, password, miner_uuid)
	VALUES ($1, $2, $3)
	`
	if _, err = db.Db.Exec(sqlStmt, c.Email, string(hashedPassword), u); err != nil {
		app.Sugar.Errorw("failed to insert miner",
			"url", r.URL,
			"err", err.Error(),
			"email", c.Email,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	app.Sugar.Infof("Miner %s successfully added!", c.Email)
	return nil
}
