package miner

import (
	"log"
	"net/http"
)

// PostOutput receives job output from miner
func PostOutput(w http.ResponseWriter, r *http.Request) {
	log.Printf("Run!\n")
	return
}
