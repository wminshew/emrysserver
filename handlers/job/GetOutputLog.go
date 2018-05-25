package job

import (
	// "github.com/wminshew/check"
	"log"
	"net/http"
)

// GetOutputLog streams the miner's container execution to the user
func GetOutputLog(w http.ResponseWriter, r *http.Request) {
	log.Printf("job.GetOutputLog!")
	_, _ = w.Write([]byte("job.GetOutputLog!\n"))
}
