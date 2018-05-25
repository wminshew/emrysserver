package miner

import (
	"log"
	"net/http"
)

// PostOutputLog receives job output from miner
func PostOutputLog(w http.ResponseWriter, r *http.Request) {
	log.Printf("Run!\n")
	return
}
