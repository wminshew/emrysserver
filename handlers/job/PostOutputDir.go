package job

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/pkg/app"
	"io"
	"net/http"
	"path"
)

// PostOutputDir receives the miner's container execution for the user
func PostOutputDir(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	jID := vars["jID"]
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		app.Sugar.Errorw("failed to parse job ID",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "Error parsing job ID"}
	}

	pipe := getDirPipe(jUUID)

	pw := pipe.pw
	tee := io.TeeReader(r.Body, pw)

	ctx := r.Context()
	p := path.Join("job", jID, "output", "dir")
	obj := bkt.Object(p)
	ow := obj.NewWriter(ctx)

	_, err = io.Copy(ow, tee)
	if err != nil {
		app.Sugar.Errorw("failed to copy tee reader to cloud storage object writer",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "Internal error"}
	}

	if err = ow.Close(); err != nil {
		app.Sugar.Errorw("failed to close cloud storage object writer",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "Internal error"}
	}
	if err = pw.Close(); err != nil {
		app.Sugar.Errorw("failed to close pipe writer",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "Internal error"}
	}

	sqlStmt := `
	UPDATE jobs
	SET (completed_at, active) = (NOW(), false)
	WHERE job_uuid = $1
	`
	_, err = db.Db.Exec(sqlStmt, jID)
	if err != nil {
		app.Sugar.Errorw("failed to update job",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "Internal error"}
	}
	sqlStmt = `
	UPDATE statuses
	SET (output_dir_posted) = ($1)
	WHERE job_uuid = $2
	`
	_, err = db.Db.Exec(sqlStmt, true, jID)
	if err != nil {
		app.Sugar.Errorw("failed to update job status",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "Internal error"}
	}

	return nil
}
