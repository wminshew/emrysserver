package miner

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/check"
	"github.com/wminshew/emrysserver/db"
	"log"
	"math"
	"sync"
	"time"
)

type auction struct {
	jobID  uuid.UUID
	winBid uuid.UUID
	mux    sync.Mutex
}

const (
	// To change auction length, must also change bid_late function within DB
	auctionLength = 5 * time.Second
)

func newAuction(jID uuid.UUID) *auction {
	return &auction{
		jobID: jID,
	}
}

// TODO: auction running should be a separate microservice
func (a *auction) run(p *pool) {
	p.auctions[a.jobID] = a
	defer func() {
		// TODO: re-factor so I don't have to manually manage this memory..?
		time.Sleep(auctionLength)
		delete(p.auctions, a.jobID)
		// a = nil
	}()
	a.mux.Lock()
	defer a.mux.Unlock()
	time.Sleep(auctionLength)

	sqlStmt := `
	SELECT bid_uuid, min_rate
	FROM bids
	WHERE job_uuid = $1
		AND late = false
	`
	rows, err := db.Db.Query(sqlStmt, a.jobID)
	if err != nil {
		log.Printf("Error selecting bids from db for job %v: %v\n", a.jobID, err)
		return
	}
	defer check.Err(rows.Close)
	n := 0
	winRate := math.Inf(1)
	payRate := math.Inf(1)
	for rows.Next() {
		var bidUUID uuid.UUID
		var bidMinRate float64
		n++
		if err = rows.Scan(&bidUUID, &bidMinRate); err != nil {
			log.Printf("Error scanning db bids: %v\n", err)
			return
		}
		if bidMinRate < winRate {
			a.winBid = bidUUID
			payRate = winRate
			winRate = bidMinRate
		} else if bidMinRate < payRate {
			payRate = bidMinRate
		}
	}
	if n == 0 {
		log.Printf("No bids received.\n")
		return
	} else if n == 1 {
		payRate = winRate
	}
	log.Printf("%d bid(s) received.\n", n)
	log.Printf("Winning bid: %v\n", a.winBid)
	log.Printf("Pay Rate: %v\n", payRate)

	sqlStmt = `
	UPDATE jobs
	SET (win_bid_uuid, pay_rate) = ($1, $2)
	WHERE job_uuid = $3
	`
	_, err = db.Db.Exec(sqlStmt, a.winBid, payRate, a.jobID)
	if err != nil {
		log.Printf("Error inserting winning bid info into jobs table: %v\n", err)
		return
	}
}

func (a *auction) winner() uuid.UUID {
	a.mux.Lock()
	defer a.mux.Unlock()
	return a.winBid
}
