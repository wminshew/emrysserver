package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// InsertJob inserts a new job, status, and payment into the db
func InsertJob(r *http.Request, uUUID, jUUID uuid.UUID) *app.Error {
	ctx := r.Context()
	tx, txerr := db.BeginTx(ctx, nil)
	if message, err := func() (string, error) {
		if txerr != nil {
			return "failed to begin tx", txerr
		}
		sqlStmt := `
	INSERT INTO jobs (job_uuid, user_uuid, active)
	VALUES ($1, $2, $3)
	`
		if _, err := tx.Exec(sqlStmt, jUUID, uUUID, true); err != nil {
			return "failed to insert job", err
		}

		sqlStmt = `
	INSERT INTO payments (job_uuid, user_paid, miner_paid)
	VALUES ($1, $2, $3)
	`
		if _, err := tx.Exec(sqlStmt, jUUID, false, false); err != nil {
			return "failed to insert payment", err
		}
		sqlStmt = `
	INSERT INTO statuses (job_uuid, user_data_stored,
	image_built, auction_completed,
	image_downloaded, data_downloaded,
	output_log_posted, output_dir_posted)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
		if _, err := tx.Exec(sqlStmt, jUUID, false, false, false, false, false, false, false); err != nil {
			return "failed to insert status", err
		}

		if err := tx.Commit(); err != nil {
			return "failed to commit tx", err
		}

		return "", nil
	}(); err != nil {
		pqErr := err.(*pq.Error)
		log.Sugar.Errorw(message,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
			"pq_sev", pqErr.Severity,
			"pq_code", pqErr.Code,
			"pq_detail", pqErr.Detail,
		)
		if txerr == nil {
			if err := tx.Rollback(); err != nil {
				log.Sugar.Errorf("Error rolling tx back job %v: %v\n", jUUID, err)
			}
		}
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	return nil
}
