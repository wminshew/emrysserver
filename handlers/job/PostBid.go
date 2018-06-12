package job

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/db"
	"log"
	"net/http"
)

// PostBid accepts a job.Bid from miner and adds it to the bids table
func PostBid(w http.ResponseWriter, r *http.Request) {
	b := &job.Bid{}
	err := json.NewDecoder(r.Body).Decode(b)
	if err != nil {
		log.Printf("Error decoding json: %v\n", err)
		http.Error(w, "Error parsing json body", http.StatusBadRequest)
		return
	}
	b.ID = uuid.NewV4()

	vals := r.URL.Query()
	mIDs, ok := vals["mID"]
	if !ok {
		log.Printf("Error finding mID query value: %v\n", err)
		http.Error(w, "Error pulling valid miner UUID from jwt. Please login again.", http.StatusBadRequest)
		return
	}
	mID := mIDs[0]

	vars := mux.Vars(r)
	jID := vars["jID"]
	b.JobID, err = uuid.FromString(jID)
	if err != nil {
		log.Printf("Error parsing job ID: %v\n", err)
		http.Error(w, "Error parsing job ID in path", http.StatusBadRequest)
		return
	}

	b.MinerID, err = uuid.FromString(mID)
	if err != nil {
		log.Printf("miner_uuid in request context corrupted\n")
		http.Error(w, "Unable to retrieve valid uuid from jwt. Please login again.", http.StatusInternalServerError)
		return
	}

	a, ok := auctions[b.JobID]
	if !ok {
		b.Late = false
	} else {
		b.Late = a.lateBid()
	}
	sqlStmt := `
	INSERT INTO bids (bid_uuid, job_uuid, miner_uuid, min_rate, late)
	VALUES ($1, $2, $3, $4, $5)
	`
	_, err = db.Db.Exec(sqlStmt, b.ID, b.JobID, b.MinerID, b.MinRate, b.Late)
	if err != nil {
		log.Printf("Error inserting bid %v for job %v: %v\n", b.ID, b.JobID, err)
		http.Error(w, "Your bid was not accepted.", http.StatusInternalServerError)
		return
	}
	log.Printf("Bid: %+v\n", b)

	if b.Late {
		_, err = w.Write([]byte("Your bid was late.\n"))
		if err != nil {
			log.Printf("Error writing response: %v\n", err)
			http.Error(w, "Error writing response.", http.StatusInternalServerError)
		}
		return
	}

	winbid := a.winBid()
	if !uuid.Equal(winbid, b.ID) {
		bidNotSelected := fmt.Sprintf("Your bid for job %v was not selected.\n", b.JobID)
		_, err = w.Write([]byte(bidNotSelected))
		if err != nil {
			log.Printf("Error writing bid response: %v\n", err)
		}
		return
	}

	w.Header().Set("Winner", "True")
}
