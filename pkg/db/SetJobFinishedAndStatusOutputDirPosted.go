package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetJobFinishedAndStatusOutputDirPosted sets job completed (and inactive) and status for job jUUID
func SetJobFinishedAndStatusOutputDirPosted(r *http.Request, jUUID uuid.UUID) *app.Error {
	ctx := r.Context()
	tx, txerr := db.BeginTx(ctx, nil)
	if message, err := func() (string, error) {
		if txerr != nil {
			return "failed to begin tx", txerr
		}

		sqlStmt := `
	UPDATE jobs
	SET (completed_at, active) = (NOW(), false)
	WHERE job_uuid = $1
	`
		if _, err := tx.Exec(sqlStmt, jUUID); err != nil {
			return "failed to update job winner", err
		}

		sqlStmt = `
	UPDATE statuses
	SET (output_dir_posted) = ($1)
	WHERE job_uuid = $2
	`
		if _, err := tx.Exec(sqlStmt, true, jUUID); err != nil {
			return "failed to update job status", err
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
			"jUUID", jUUID,
			"pq_sev", pqErr.Severity,
			"pq_code", pqErr.Code,
			"pq_detail", pqErr.Detail,
		)
		if txerr == nil {
			if err := tx.Rollback(); err != nil {
				log.Sugar.Errorf("Error rolling tx back job %v: %v\n", jUUID, err)
			}
		}
		if err := SetJobInactive(r, jUUID); err != nil {
			log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
		}
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	return nil
}
