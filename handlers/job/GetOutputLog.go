package job

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/flushwriter"
	"io"
	"log"
	"net/http"
)

// GetOutputLog streams the miner's container execution to the user
func GetOutputLog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jID := vars["jID"]
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		log.Printf("Error converting jID %s to uuid: %v\n", jID, err)
		http.Error(w, "Internal Error.", http.StatusInternalServerError)
		return
	}

	if outputPipes[jUUID] == nil {
		pr, pw := io.Pipe()
		outputPipes[jUUID] = &pipe{
			pr: pr,
			pw: pw,
		}
	}

	fw := flushwriter.New(w)
	pr := outputPipes[jUUID].pr
	_, _ = io.Copy(fw, pr)
	err = pr.Close()
	if err != nil {
		log.Printf("Error closing output pipe: %v\n", err)
	}
	delete(outputPipes, jUUID)
}
