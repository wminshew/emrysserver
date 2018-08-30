package db

import (
	"database/sql"
	"errors"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

var (
	// ErrUnauthorizedUser is returned when the given email address doesn't exist in the db
	ErrUnauthorizedUser = errors.New("unauthorized user")
)

// GetUserUUIDAndPassword returns user uuid and hashed password of the email address, if they exist
func GetUserUUIDAndPassword(r *http.Request, email string) (uuid.UUID, string, error) {
	storedC := &creds.User{}
	uUUID := uuid.UUID{}
	sqlStmt := `
	SELECT password, user_uuid
	FROM users
	WHERE user_email=$1
	`
	if err := db.QueryRow(sqlStmt, email).Scan(&storedC.Password, &uUUID); err != nil {
		if err == sql.ErrNoRows {
			log.Sugar.Infow("unauthorized user",
				"url", r.URL,
				"email", email,
			)
			return uuid.UUID{}, "", ErrUnauthorizedUser
		}
		message := "error querying for user uuid and password"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"url", r.URL,
				"err", err.Error(),
				"email", email,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"url", r.URL,
				"err", err.Error(),
				"email", email,
			)
		}
		return uuid.UUID{}, "", err
	}
	return uUUID, storedC.Password, nil
}
