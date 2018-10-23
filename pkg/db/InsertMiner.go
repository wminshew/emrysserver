package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// InsertMiner inserts a new miner into the db
func InsertMiner(r *http.Request, email, hashedPassword string, mUUID uuid.UUID) error {
	sqlStmt := `
	INSERT INTO miners (miner_email, password, miner_uuid)
	VALUES ($1, $2, $3)
	`
	if _, err := db.Exec(sqlStmt, email, hashedPassword, mUUID); err != nil {
		message := "error inserting miner"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"mID", mUUID,
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
				"mID", mUUID,
				"email", email,
			)
		}
		return err
	}

	return nil
}
