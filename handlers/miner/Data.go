package miner

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/check"
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

	// TODO: use claims? or something? to specify correct path...
	dataPath := filepath.Join("job-upload", jID, "data.tar.gz")
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
}
