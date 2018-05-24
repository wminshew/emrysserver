package job

import (
	"log"
	"net/http"
)

// PostOutput receives the miner's container execution for the user
func PostOutput(w http.ResponseWriter, r *http.Request) {
	log.Printf("job.PostOutput!")
}
