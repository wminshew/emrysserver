package miner

import (
	"log"
	"net/http"
)

// Data ...
func Data(w http.ResponseWriter, r *http.Request) {
	log.Printf("Data!\n")
	return
}
