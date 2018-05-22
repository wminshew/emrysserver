package miner

import (
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/db"
	"log"
	"net/http"
	"time"
)

// Bid accepts a job.Bid from miner and adds it to the bids table
func Bid(w http.ResponseWriter, r *http.Request) {
	b := &job.Bid{}
	err := json.NewDecoder(r.Body).Decode(b)
	if err != nil {
		log.Printf("Error decoding json: %v\n", err)
		http.Error(w, "Error parsing json body", http.StatusBadRequest)
		return
	}
	b.ID = uuid.NewV4()

	vars := mux.Vars(r)
	jID := vars["jID"]
	b.JobID, err = uuid.FromString(jID)
	if err != nil {
		log.Printf("Error parsing job ID: %v\n", err)
		http.Error(w, "Error parsing job ID in path", http.StatusBadRequest)
		return
	}

	ctxKey := contextKey("miner_uuid")
	mUUID, ok := r.Context().Value(ctxKey).(uuid.UUID)
	if !ok {
		log.Printf("miner_uuid in request context corrupted\n")
		http.Error(w, "Unable to retrieve valid uuid from jwt. Please login again.", http.StatusInternalServerError)
		return
	}
	b.MinerID = mUUID

	sqlStmt := `
	INSERT INTO bids (bid_uuid, job_uuid, miner_uuid, min_rate)
	VALUES ($1, $2, $3, $4)
	RETURNING late
	`
	err = db.Db.QueryRow(sqlStmt, b.ID, b.JobID, b.MinerID, b.MinRate).Scan(&b.Late)
	if err != nil {
		log.Printf("Error inserting bid into db: %v\n", err)
		http.Error(w, "Your bid was not accepted.", http.StatusInternalServerError)
		return
	}
	log.Printf("Bid: %+v\n", b)

	if b.Late {
		log.Printf("Late bid: %v\n", b.ID)
		return
	}

	if Pool.auctions[b.JobID] == nil {
		return
	}
	winbid := Pool.auctions[b.JobID].winner()
	if !uuid.Equal(winbid, b.ID) {
		return
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * 1).Unix(),
		"iss": "bid.service",
		"iat": time.Now().Unix(),
		"sub": b.JobID,
	})

	tString, err := t.SignedString([]byte(secret))
	if err != nil {
		log.Printf("Internal error: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	resp := job.BidResp{
		Token: tString,
	}
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding JSON response: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
