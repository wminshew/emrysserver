package db

import (
	"context"
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"time"
)

// https://github.com/lib/pq/blob/master/error.go#L78
const errCheckViolation = "23514" // "check_violation"

// SetJobFailed sets job failed_at and active=false for job jUUID
func SetJobFailed(jUUID uuid.UUID, baseMinerPenalty float64) error {
	ctx := context.Background()
	tx, txerr := db.BeginTx(ctx, nil)
	if message, err := func() (string, error) {
		if txerr != nil {
			return errBeginTx, txerr
		}
		mUUID := uuid.UUID{}
		createdAt := time.Time{}
		failedAt := pq.NullTime{}
		rate := sql.NullFloat64{}
		sqlStmt := `
		UPDATE jobs j
		SET failed_at = NOW(),
		active = false
		FROM miners m, bids b
		WHERE j.uuid = $1 AND
			j.failed_at IS NULL AND
			b.uuid = j.win_bid_uuid AND
			m.uuid = b.miner_uuid
		RETURNING m.uuid, j.created_at, j.failed_at, j.rate
		`
		if err := db.QueryRow(sqlStmt, jUUID).Scan(&mUUID, &createdAt, &failedAt, &rate); err != nil {
			return "error updating jobs failed_at, active", err
		}

		if failedAt.Valid {
			if rate.Valid {
				amt := baseMinerPenalty + rate.Float64*failedAt.Time.Sub(createdAt).Hours()
				sqlStmt = `
				UPDATE accounts a
				SET balance = balance - $2
				WHERE a.uuid = $1
				`
				if _, err := db.Exec(sqlStmt, mUUID, amt); err != nil {
					return "error updating miner accounts balance", err
				}

				sqlStmt = `
				UPDATE payments p
				SET processed = $2,
				miner_penalty = $3
				WHERE p.job_uuid = $1 AND
					p.processed IS NULL
				`
				if _, err := db.Exec(sqlStmt, jUUID, failedAt.Time, -amt); err != nil {
					return "error updating payments processed", err
				}
			} else {
				// shouldn't happen
				log.Sugar.Errorf("null rate")
			}
		} else {
			// shouldn't happen
			log.Sugar.Errorf("null failedAt")
		}

		if err := tx.Commit(); err != nil {
			return errCommitTx, err
		}

		return "", nil
	}(); err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok {
			if pqErr.Code == errCheckViolation {
				log.Sugar.Infow("Job already ended, not updating failed_at",
					"jID", jUUID,
				)
			} else {
				log.Sugar.Errorw(message,
					"err", err.Error(),
					"jID", jUUID,
					"pq_sev", pqErr.Severity,
					"pq_code", pqErr.Code,
					"pq_detail", pqErr.Detail,
				)
			}
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"jID", jUUID,
			)
		}
		if txerr == nil {
			if err := tx.Rollback(); err != nil {
				log.Sugar.Errorf("Error rolling tx back: %v", err)
			}
		}
		return err
	}

	return nil
}
