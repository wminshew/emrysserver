package miner

import (
	// "database/sql"
	"encoding/json"
	// "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/db"
	"log"
	"net/http"
	// "os"
	// "time"
)

// type bidResponse struct {
// 	Token string `json:"token"`
// }

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

	if _, err = db.Db.Query("INSERT INTO bids (bid_uuid, job_uuid, miner_uuid, min_rate) VALUES ($1, $2, $3, $4)",
		b.ID, b.JobID, b.MinerID, b.MinRate); err != nil {
		log.Printf("Error inserting bid into db: %v\n", err)
		http.Error(w, "Your bid was not accepted.", http.StatusInternalServerError)
		return
	}

	return
	// hold response until server decides which bid wins?
	// token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
	// 	"exp":   time.Now().Add(time.Hour * 24).Unix(),
	// 	"iss":   "auth.service",
	// 	"iat":   time.Now().Unix(),
	// 	"email": storedC.Email,
	// 	"sub":   u,
	// })
	//
	// tokenString, err := token.SignedString([]byte(secret))
	// if err != nil {
	// 	log.Printf("Internal error: %v\n", err)
	// 	http.Error(w, "Internal error.", http.StatusInternalServerError)
	// 	return
	// }
	//
	// response := tokenResponse{
	// 	Token: tokenString,
	// }
	// if err = json.NewEncoder(w).Encode(response); err != nil {
	// 	log.Printf("Error encoding JSON response: %v\n", err)
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
}
