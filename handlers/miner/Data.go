package miner

import (
	"log"
	"net/http"
)

// Data sends the data.tar.gz, if it exists, associated with job jID to the miner
func Data(w http.ResponseWriter, r *http.Request) {
	log.Printf("Data!\n")
	return
}
