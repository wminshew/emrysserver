package user

import (
	"github.com/wminshew/emrysserver/pkg/flushwriter"
	"log"
	"net/http"
)

// GetOutputLog streams job output to user
func GetOutputLog(w http.ResponseWriter, r *http.Request) {
	log.Printf("user.GetOutputLog!\n")

	fw := flushwriter.New(w)

	_, err := fw.Write([]byte("Running image...\n"))
	if err != nil {
		log.Printf("Error writing to flushWriter: %v\n", err)
	}

	// TODO: Pipe output back to user
	// TODO: use third 'job' server?
}
