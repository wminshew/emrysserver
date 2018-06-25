package miner

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/pkg/app"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Data sends the data.tar.gz, if it exists, associated with job jID to the miner
func Data(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	jID := vars["jID"]
	mID := vars["mID"]

	inputDir := filepath.Join("job", jID, "input")
	defer check.Err(func() error { return os.RemoveAll(inputDir) })
	dataPath := filepath.Join(inputDir, "data")
	dataFile, err := os.Open(dataPath)
	if err != nil {
		app.Sugar.Errorw("failed to open data file",
			"url", r.URL,
			"path", dataPath,
			"err", err.Error(),
			"jID", jID,
			"mID", mID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	defer check.Err(dataFile.Close)

	if _, err = io.Copy(w, dataFile); err != nil {
		app.Sugar.Errorw("failed to copy data file to response writer",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
			"mID", mID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "Internal error"}
	}

	sqlStmt := `
	UPDATE statuses
	SET (data_downloaded) = ($1)
	WHERE job_uuid = $2
	`
	if _, err = db.Db.Exec(sqlStmt, true, jID); err != nil {
		app.Sugar.Errorw("failed to update job status",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
			"mID", mID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "Internal error"}
	}

	return nil
}
