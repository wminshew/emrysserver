package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// InsertMiner inserts a new miner into the db
func InsertMiner(r *http.Request, email, hashedPassword string, mUUID uuid.UUID) *app.Error {
	sqlStmt := `
	INSERT INTO miners (miner_email, password, miner_uuid)
	VALUES ($1, $2, $3)
	`
	if _, err := db.Exec(sqlStmt, email, hashedPassword, mUUID); err != nil {
		pqErr := err.(*pq.Error)
		log.Sugar.Errorw("failed to insert miner",
			"url", r.URL,
			"err", err.Error(),
			"mID", mUUID,
			"email", email,
			"pq_sev", pqErr.Severity,
			"pq_code", pqErr.Code,
			"pq_detail", pqErr.Detail,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	log.Sugar.Infof("User %s (%s) successfully added!", email, mUUID.String())
	return nil
}
