package main

import (
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

// data sends the data.tar.gz, if it exists, associated with job jID to the miner
func data(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	jID := vars["jID"]
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		log.Sugar.Errorw("failed to parse job ID",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	inputDir := filepath.Join("job", jID, "input")
	// defer app.CheckErr(r, func() error { return os.RemoveAll(inputDir) })
	dataPath := filepath.Join(inputDir, "data")
	var tee io.Reader
	var dataFile *os.File
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		// if not cached to disk, stream from cloud storage and cache
		ctx := r.Context()
		or, err := bkt.Object(dataPath).NewReader(ctx)
		if err != nil {
			log.Sugar.Errorw("failed to open cloud storage reader",
				"url", r.URL,
				"path", dataPath,
				"err", err.Error(),
				"jID", jID,
			)
if err := db.SetJobInactive(r, jUUID); err != nil {
			log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
if err := db.SetJobInactive(r, jUUID); err != nil {
			log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		if dataFile, err = os.Create(dataPath); err != nil {
			log.Sugar.Errorw("failed to create disk cache",
				"url", r.URL,
				"path", dataPath,
				"err", err.Error(),
				"jID", jID,
			)
if err := db.SetJobInactive(r, jUUID); err != nil {
			log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
if err := db.SetJobInactive(r, jUUID); err != nil {
			log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		tee = io.TeeReader(or, dataFile)
	} else {
		if dataFile, err = os.Open(dataPath); err != nil {
			log.Sugar.Errorw("failed to open data file",
				"url", r.URL,
				"path", dataPath,
				"err", err.Error(),
				"jID", jID,
			)
if err := db.SetJobInactive(r, jUUID); err != nil {
			log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
if err := db.SetJobInactive(r, jUUID); err != nil {
			log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		tee = io.TeeReader(dataFile, ioutil.Discard)
	}
	defer app.CheckErr(r, dataFile.Close)

	if _, err := io.Copy(w, tee); err != nil {
		log.Sugar.Errorw("failed to copy data file to response writer",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
if err := db.SetJobInactive(r, jUUID); err != nil {
		log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
if err := db.SetJobInactive(r, jUUID); err != nil {
		log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
		return &app.Error{Code: http.StatusInternalServerError, Message: "Internal error"}
	}

	sqlStmt := `
	UPDATE statuses
	SET (data_downloaded) = ($1)
	WHERE job_uuid = $2
	`
	if _, err := db.Db.Exec(sqlStmt, true, jID); err != nil {
		pqErr := err.(*pq.Error)
		if pqErr.Fatal() {
			log.Sugar.Fatalw("failed to update job status",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw("failed to update job status",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		}
if err := db.SetJobInactive(r, jUUID); err != nil {
		log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
if err := db.SetJobInactive(r, jUUID); err != nil {
		log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
		return &app.Error{Code: http.StatusInternalServerError, Message: "Internal error"}
	}

	return nil
}
