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
	// ErrUnauthorizedMiner is returned when the given email address doesn't exist in the db
	ErrUnauthorizedMiner = errors.New("unauthorized miner")
)

// GetMinerUUIDAndPassword returns miner uuid and hashed password of the email address, if they exist
func GetMinerUUIDAndPassword(r *http.Request, email string) (uuid.UUID, string, error) {
	storedC := &creds.Miner{}
	mUUID := uuid.UUID{}
	sqlStmt := `
	SELECT password, miner_uuid
	FROM miners
	WHERE miner_email=$1
	`
	if err := db.QueryRow(sqlStmt, email).Scan(&storedC.Password, &mUUID); err != nil {
		if err == sql.ErrNoRows {
			log.Sugar.Infow("unauthorized miner",
				"method", r.Method,
				"url", r.URL,
				"email", email,
			)
			return uuid.UUID{}, "", ErrUnauthorizedMiner
		}
		message := "error querying for miner uuid and password"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"email", email,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
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
		return uuid.UUID{}, "", err
	}
	return mUUID, storedC.Password, nil
}
