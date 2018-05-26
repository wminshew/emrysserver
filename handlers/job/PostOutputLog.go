package job

import (
	// "bufio"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"io"
	"log"
	"net/http"
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

	if outputPipes[jUUID] == nil {
		pr, pw := io.Pipe()
		outputPipes[jUUID] = &pipe{
			pr: pr,
			pw: pw,
		}
	}

	pw := outputPipes[jUUID].pw
	_, _ = io.Copy(pw, r.Body)
	err = pw.Close()
	if err != nil {
		log.Printf("Error closing output pipe: %v\n", err)
	}
}
