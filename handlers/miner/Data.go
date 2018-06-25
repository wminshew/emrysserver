package miner

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/check"
	"io"
	"io/ioutil"
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
	// defer check.Err(r, func() error { return os.RemoveAll(inputDir) })
	dataPath := filepath.Join(inputDir, "data")
	var tee io.Reader
	var dataFile *os.File
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		// if not cached to disk, stream from cloud storage and cache
		ctx := r.Context()
		or, err := bkt.Object(dataPath).NewReader(ctx)
		if err != nil {
			app.Sugar.Errorw("failed to open cloud storage reader",
				"url", r.URL,
				"path", dataPath,
				"err", err.Error(),
				"jID", jID,
				"mID", mID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		dataFile, err = os.Create(dataPath)
		if err != nil {
			app.Sugar.Errorw("failed to create disk cache",
				"url", r.URL,
				"path", dataPath,
				"err", err.Error(),
				"jID", jID,
				"mID", mID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		tee = io.TeeReader(or, dataFile)
	} else {
		dataFile, err = os.Open(dataPath)
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
		tee = io.TeeReader(dataFile, ioutil.Discard)
	}
	defer check.Err(r, dataFile.Close)

	if _, err := io.Copy(w, tee); err != nil {
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
	if _, err := db.Db.Exec(sqlStmt, true, jID); err != nil {
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
