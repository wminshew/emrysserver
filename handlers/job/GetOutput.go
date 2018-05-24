package job

import (
	"log"
	"net/http"
)

// GetOutput streams the miner's container execution to the user
func GetOutput(w http.ResponseWriter, r *http.Request) {
	log.Printf("job.GetOutput!")
}
