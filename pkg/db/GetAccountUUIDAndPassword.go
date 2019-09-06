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
	// ErrUnauthorizedAccount is returned when the given email address doesn't exist in the db
	ErrUnauthorizedAccount = errors.New("unauthorized account")
)

// GetAccountUUIDAndPassword returns account uuid and hashed password of the email address, if they exist
func GetAccountUUIDAndPassword(r *http.Request, email string) (uuid.UUID, string, bool, bool, bool, bool, bool, error) {
	aUUID := uuid.UUID{}
	storedC := &creds.Account{}
	uUUID := uuid.NullUUID{}
	mUUID := uuid.NullUUID{}
	confirmed := false
	suspended := false
	beta := false
	sqlStmt := `
	SELECT a.uuid, password, u.uuid, m.uuid, a.confirmed, a.suspended, a.beta
	FROM accounts a
	LEFT OUTER JOIN users u ON (a.uuid = u.uuid)
	LEFT OUTER JOIN miners m ON (a.uuid = m.uuid)
	WHERE a.email=$1
	`
	if err := db.QueryRow(sqlStmt, email).Scan(&aUUID, &storedC.Password, &uUUID, &mUUID, &confirmed, &suspended, &beta); err != nil {
		if err == sql.ErrNoRows {
			log.Sugar.Infow("unauthorized account -- no email in db",
				"method", r.Method,
				"url", r.URL,
				"email", email,
			)
			return uuid.UUID{}, "", false, false, false, false, false, ErrUnauthorizedAccount
		}
		message := "error querying for account by email"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"email", email,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_msg", pqErr.Message,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"email", email,
			)
		}
		return uuid.UUID{}, "", false, false, false, false, false, err
	}
	isUser := uUUID.Valid
	isMiner := mUUID.Valid
	return aUUID, storedC.Password, isUser, isMiner, confirmed, suspended, beta, nil
}
