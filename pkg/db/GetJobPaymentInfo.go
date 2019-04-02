package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"time"
)

// GetJobPaymentInfo gets rate, createdAt, completedAt, canceledAt, failedAt for job jUUID
func GetJobPaymentInfo(jUUID uuid.UUID) (float64, time.Time, time.Time, time.Time, time.Time, error) {
	rate := sql.NullFloat64{}
	createdAt := pq.NullTime{}
	completedAt := pq.NullTime{}
	canceledAt := pq.NullTime{}
	failedAt := pq.NullTime{}
	sqlStmt := `
	SELECT rate, created_at, completed_at, canceled_at, failed_at
	FROM jobs
	WHERE job_uuid = $1
	`
	if err := db.QueryRow(sqlStmt, jUUID).Scan(&rate, &createdAt, &completedAt, &canceledAt, &failedAt); err != nil {
		message := "error querying jobs payment info"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"jID", jUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"jID", jUUID,
			)
		}
		return 0, time.Time{}, time.Time{}, time.Time{}, time.Time{}, err
	}

	var rateReturn float64
	if rate.Valid {
		rateReturn = rate.Float64
	}
	createdAtReturn := time.Time{}
	if createdAt.Valid {
		createdAtReturn = createdAt.Time
	}
	completedAtReturn := time.Time{}
	if completedAt.Valid {
		completedAtReturn = completedAt.Time
	}
	canceledAtReturn := time.Time{}
	if canceledAt.Valid {
		canceledAtReturn = canceledAt.Time
	}
	failedAtReturn := time.Time{}
	if failedAt.Valid {
		failedAtReturn = failedAt.Time
	}
	return rateReturn, createdAtReturn, completedAtReturn, canceledAtReturn, failedAtReturn, nil
}
