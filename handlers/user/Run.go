package user

import (
	"github.com/wminshew/emrysserver/pkg/flushwriter"
	"log"
	"net/http"
)

// Run ...
func Run(w http.ResponseWriter, r *http.Request) {
	log.Printf("Run!\n")

	fw := flushwriter.New(w)

	_, err := fw.Write([]byte("Running image...\n"))
	if err != nil {
		log.Printf("Error writing to flushWriter: %v\n", err)
	}

	// TODO: Pipe output back to user
	// TODO: use third 'job' server?
}
