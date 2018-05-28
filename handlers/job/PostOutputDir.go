package job

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/db"
	"io"
	"log"
	"net/http"
)

// PostOutputDir receives the miner's container execution for the user
func PostOutputDir(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jID := vars["jID"]
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		log.Printf("Error converting jID %s to uuid: %v\n", jID, err)
		http.Error(w, "Internal Error.", http.StatusInternalServerError)
		return
	}

	if outputDir[jUUID] == nil {
		pr, pw := io.Pipe()
		outputDir[jUUID] = &pipe{
			pr: pr,
			pw: pw,
		}
	}

	pw := outputDir[jUUID].pw
	_, _ = io.Copy(pw, r.Body)
	err = pw.Close()
	if err != nil {
		log.Printf("Error closing output pipe: %v\n", err)
	}

	go func() {
		sqlStmt := `
		UPDATE jobs
		SET (completed_at) = (NOW())
		WHERE job_uuid = $1
		`
		_, err = db.Db.Exec(sqlStmt, jID)
		if err != nil {
			log.Printf("Error inserting finished_at into jobs table for job %v: %v\n", jID, err)
			return
		}
	}()
}
