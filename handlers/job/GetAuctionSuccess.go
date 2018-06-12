package job

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/db"
	"log"
	"net/http"
	"time"
)

type auctionSuccess struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// GetAuctionSuccess returns whether an auction is successful
func GetAuctionSuccess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jID := vars["jID"]
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		log.Printf("Error parsing job ID: %v\n", err)
		http.Error(w, "Error parsing job ID in path", http.StatusBadRequest)
		return
	}

	time.Sleep(duration)

	a, ok := auctions[jUUID]
	if ok {
		winBid := a.winBid()
		if uuid.Equal(winBid, uuid.Nil) {
			err = json.NewEncoder(w).Encode(auctionSuccess{
				Success: false,
				Error:   "No bids received!",
			})
			if err != nil {
				log.Printf("Error encoding json auctionSuccess: %v\n", err)
				http.Error(w, "Internal error!", http.StatusInternalServerError)
				return
			}
		} else {
			err = json.NewEncoder(w).Encode(auctionSuccess{
				Success: true,
			})
			if err != nil {
				log.Printf("Error encoding json auctionSuccess: %v\n", err)
				http.Error(w, "Internal error!", http.StatusInternalServerError)
				return
			}
		}
	} else {
		var success bool
		sqlStmt := `
		SELECT auction_completed
		FROM statuses
		WHERE job_uuid = $1
		`
		err = db.Db.QueryRow(sqlStmt, jID).Scan(&success)
		if err != nil {
			log.Printf("Error querying job auction status: %v\n", err)
			err = json.NewEncoder(w).Encode(auctionSuccess{
				Success: false,
				Error:   "Internal error!",
			})
			if err != nil {
				log.Printf("Error encoding json auctionSuccess: %v\n", err)
				http.Error(w, "Internal error!", http.StatusInternalServerError)
				return
			}
		}
		if success {
			err = json.NewEncoder(w).Encode(auctionSuccess{
				Success: true,
			})
			if err != nil {
				log.Printf("Error encoding json auctionSuccess: %v\n", err)
				http.Error(w, "Internal error!", http.StatusInternalServerError)
				return
			}
		} else {
			err = json.NewEncoder(w).Encode(auctionSuccess{
				Success: false,
				Error:   "Internal error!",
			})
			if err != nil {
				log.Printf("Error encoding json auctionSuccess: %v\n", err)
				http.Error(w, "Internal error!", http.StatusInternalServerError)
				return
			}
		}
	}
}
