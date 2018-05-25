package job

import (
	"log"
	"net/http"
)

// GetOutputLog streams the miner's container execution to the user
func GetOutputLog(w http.ResponseWriter, r *http.Request) {
	log.Printf("job.GetOutputLog!")
}
