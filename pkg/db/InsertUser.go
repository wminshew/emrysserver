package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// InsertUser inserts a new user into the db
func InsertUser(r *http.Request, email, hashedPassword string, uUUID uuid.UUID) *app.Error {
	sqlStmt := `
	INSERT INTO users (user_email, password, user_uuid)
	VALUES ($1, $2, $3)
	`
	if _, err := Db.Exec(sqlStmt, email, hashedPassword, uUUID); err != nil {
		pqErr := err.(*pq.Error)
		log.Sugar.Errorw("failed to insert user",
			"url", r.URL,
			"err", err.Error(),
			"uID", uUUID,
			"email", email,
			"pq_sev", pqErr.Severity,
			"pq_code", pqErr.Code,
			"pq_detail", pqErr.Detail,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	log.Sugar.Infof("User %s (%s) successfully added!", email, uUUID.String())
	return nil
}
