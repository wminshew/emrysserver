package job

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/db"
	"io"
	"log"
	"net/http"
	"path"
)

// PostOutputLog receives the miner's container execution for the user
func PostOutputLog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jID := vars["jID"]
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		log.Printf("Error converting jID %s to uuid: %v\n", jID, err)
		http.Error(w, "Internal Error.", http.StatusInternalServerError)
		return
	}

	// TODO: technically I think this is a race condition between PostOutputLog and GetOutputLog
	// how can I make it idempotent?
	if outputLog[jUUID] == nil {
		pr, pw := io.Pipe()
		outputLog[jUUID] = &pipe{
			pr: pr,
			pw: pw,
		}
	}

	pw := outputLog[jUUID].pw
	tee := io.TeeReader(r.Body, pw)

	ctx := context.Background()
	p := path.Join("job", jID, "output", "log")
	obj := outputBkt.Object(p)
	ow := obj.NewWriter(ctx)

	_, err = io.Copy(ow, tee)
	if err != nil {
		log.Printf("Error copying request body to cloud storage object: %v\n", err)
	}

	if err = ow.Close(); err != nil {
		log.Printf("Error closing cloud storage object writer: %v\n", err)
	}
	if err = pw.Close(); err != nil {
		log.Printf("Error closing output pipe: %v\n", err)
	}

	go func() {
		sqlStmt := `
		UPDATE statuses
		SET (output_log_posted) = ($1)
		WHERE job_uuid = $2
		`
		_, err = db.Db.Exec(sqlStmt, true, jID)
		if err != nil {
			log.Printf("Error updating job status (output_log_posted): %v\n", err)
			return
		}
	}()
}
