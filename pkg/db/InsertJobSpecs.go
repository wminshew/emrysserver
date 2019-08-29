package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// InsertJobSpecs inserts job specs into db
func InsertJobSpecs(r *http.Request, jUUID uuid.UUID, specs *job.Specs) error {
	sqlStmt := `
	INSERT INTO requirements (job_uuid, rate, gpu, ram, disk, pcie)
	VALUES ($1, $2, $3, $4, $5, $6)
	`
	if _, err := db.Exec(sqlStmt, jUUID, specs.Rate, specs.GPU, specs.RAM, specs.Disk, specs.Pcie); err != nil {
		message := "error inserting job requirements"
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
		return err
	}
	return nil
}
