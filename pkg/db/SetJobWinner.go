package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetJobWinnerAndAuctionStatus sets the winBid UUID, pay rate, and status for job jUUID
func SetJobWinnerAndAuctionStatus(r *http.Request, jUUID, wbUUID uuid.UUID, payRate float64) *app.Error {
	ctx := r.Context()
	tx, txerr := Db.BeginTx(ctx, nil)
	if message, err := func() (string, error) {
		if txerr != nil {
			return "failed to begin tx", txerr
		}

		sqlStmt := `
		UPDATE jobs
		SET (win_bid_uuid, pay_rate) = ($1, $2)
		WHERE job_uuid = $3
		`
		if _, err := tx.Exec(sqlStmt, wbUUID, payRate, jUUID); err != nil {
			return "failed to update job winner", err
		}

		sqlStmt = `
		UPDATE statuses
		SET (auction_completed) = ($1)
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
			"jID", jUUID,
			"pq_sev", pqErr.Severity,
			"pq_code", pqErr.Code,
			"pq_detail", pqErr.Detail,
		)
		if txerr == nil {
			_ = tx.Rollback()
		}
		_ = SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	return nil
}
