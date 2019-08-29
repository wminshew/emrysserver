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
	tx, txerr := db.BeginTx(ctx, nil)
	if message, err := func() (string, error) {
		if txerr != nil {
			return errBeginTx, txerr
		}

		sqlStmt := `
		UPDATE jobs
		SET (win_bid_uuid, rate) = ($1, $2)
		WHERE uuid = $3 AND
			win_bid_uuid IS NULL AND
			rate IS NULL
		`
		if _, err := tx.Exec(sqlStmt, wbUUID, payRate, jUUID); err != nil {
			return "error updating job winner", err
		}

		sqlStmt = `
		UPDATE statuses
		SET auction_completed = NOW()
		WHERE job_uuid = $1 AND
			auction_completed IS NULL
		`
		if _, err := tx.Exec(sqlStmt, jUUID); err != nil {
			return "error updating job status", err
		}

		if err := tx.Commit(); err != nil {
			return errCommitTx, err
		}

		return "", nil
	}(); err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_name", pqErr.Name,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
		}
		if txerr == nil {
			if err := tx.Rollback(); err != nil {
				log.Sugar.Errorf("Error rolling tx back: %v", err)
			}
		}
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	return nil
}
