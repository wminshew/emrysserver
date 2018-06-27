package job

import (
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/check"
	"io"
	"io/ioutil"
	"net/http"
	"path"
)

// PostOutputLog receives the miner's container execution for the user
func PostOutputLog(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	jID := vars["jID"]
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		app.Sugar.Errorw("failed to parse job ID",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	var tee io.Reader
	if pipe, ok := logPipes[jUUID]; ok {
		tee = io.TeeReader(r.Body, pipe.w)
		defer check.Err(r, pipe.w.Close)
	} else {
		tee = io.TeeReader(r.Body, ioutil.Discard)
	}

	ctx := r.Context()
	p := path.Join("job", jID, "output", "log")
	obj := bkt.Object(p)
	ow := obj.NewWriter(ctx)

	if _, err = io.Copy(ow, tee); err != nil {
		app.Sugar.Errorw("failed to copy tee reader to cloud storage object writer",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if err = ow.Close(); err != nil {
		app.Sugar.Errorw("failed to close cloud storage object writer",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	sqlStmt := `
	UPDATE statuses
	SET (output_log_posted) = ($1)
	WHERE job_uuid = $2
	`
	if _, err = db.Db.Exec(sqlStmt, true, jID); err != nil {
		pqErr := err.(*pq.Error)
		if pqErr.Fatal() {
			app.Sugar.Fatalw("failed to update job status",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			app.Sugar.Errorw("failed to update job status",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		}
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
