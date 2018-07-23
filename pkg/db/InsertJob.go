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
	tx, err := Db.BeginTx(ctx, nil)
	if err != nil {
		pqErr := err.(*pq.Error)
		log.Sugar.Errorw("failed to begin tx",
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
			"pq_sev", pqErr.Severity,
			"pq_code", pqErr.Code,
			"pq_detail", pqErr.Detail,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	sqlStmt := `
	INSERT INTO jobs (job_uuid, user_uuid, active)
	VALUES ($1, $2, $3)
	`
	if _, err = tx.Exec(sqlStmt, jUUID, uUUID, true); err != nil {
		pqErr := err.(*pq.Error)
		log.Sugar.Errorw("failed to insert job",
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
			"pq_sev", pqErr.Severity,
			"pq_code", pqErr.Code,
			"pq_detail", pqErr.Detail,
		)
		_ = tx.Rollback()
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	sqlStmt = `
	INSERT INTO payments (job_uuid, user_paid, miner_paid)
	VALUES ($1, $2, $3)
	`
	if _, err = tx.Exec(sqlStmt, jUUID, false, false); err != nil {
		pqErr := err.(*pq.Error)
		log.Sugar.Errorw("failed to insert payment",
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
			"pq_sev", pqErr.Severity,
			"pq_code", pqErr.Code,
			"pq_detail", pqErr.Detail,
		)
		_ = tx.Rollback()
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	sqlStmt = `
	INSERT INTO statuses (job_uuid, user_data_stored,
	image_built, auction_completed,
	image_downloaded, data_downloaded,
	output_log_posted, output_dir_posted)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	if _, err = tx.Exec(sqlStmt, jUUID, false, false, false, false, false, false, false); err != nil {
		pqErr := err.(*pq.Error)
		log.Sugar.Errorw("failed to insert status",
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
			"pq_sev", pqErr.Severity,
			"pq_code", pqErr.Code,
			"pq_detail", pqErr.Detail,
		)
		_ = tx.Rollback()
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if err = tx.Commit(); err != nil {
		pqErr := err.(*pq.Error)
		log.Sugar.Errorw("failed to commit tx",
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
			"pq_sev", pqErr.Severity,
			"pq_code", pqErr.Code,
			"pq_detail", pqErr.Detail,
		)
		_ = tx.Rollback()
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
