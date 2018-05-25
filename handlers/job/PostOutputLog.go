package job

import (
	"log"
	"net/http"
)

// PostOutputLog receives the miner's container execution for the user
func PostOutputLog(w http.ResponseWriter, r *http.Request) {
	log.Printf("job.PostOutputLog!")
}
