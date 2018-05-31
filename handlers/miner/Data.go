package miner

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrysserver/db"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// Data sends the data.tar.gz, if it exists, associated with job jID to the miner
func Data(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jID := vars["jID"]

	jobDir := filepath.Join("job-upload", jID)
	defer check.Err(func() error { return os.RemoveAll(jobDir) })
	dataPath := filepath.Join(jobDir, "data.tar.gz")
	dataFile, err := os.Open(dataPath)
	if err != nil {
		log.Printf("Error opening data file: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer check.Err(dataFile.Close)

	_, err = io.Copy(w, dataFile)
	if err != nil {
		log.Printf("Error copying data dir to response writer: %v\n", err)
		return
	}

	go func() {
		sqlStmt := `
		UPDATE statuses
		SET (data_downloaded) = ($1)
		WHERE job_uuid = $2
		`
		_, err = db.Db.Exec(sqlStmt, true, jID)
		if err != nil {
			log.Printf("Error updating job status (data_downloaded): %v\n", err)
			return
		}
	}()
}
