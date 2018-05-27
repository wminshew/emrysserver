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

	if outputLog[jUUID] == nil {
		pr, pw := io.Pipe()
		outputLog[jUUID] = &pipe{
			pr: pr,
			pw: pw,
		}
	}

	fw := flushwriter.New(w)
	pr := outputLog[jUUID].pr
	_, _ = io.Copy(fw, pr)
	delete(outputLog, jUUID)
}
