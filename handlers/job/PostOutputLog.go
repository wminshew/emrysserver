package job

import (
	"bufio"
	"log"
	"net/http"
)

// PostOutputLog receives the miner's container execution for the user
func PostOutputLog(w http.ResponseWriter, r *http.Request) {
	log.Printf("job.PostOutputLog!\n")

	scanner := bufio.NewScanner(r.Body)
	for scanner.Scan() {
		log.Println(scanner.Text())
	}
}
