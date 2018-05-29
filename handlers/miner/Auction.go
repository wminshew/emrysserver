package miner

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/check"
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
		// Best case = event-triggered from DB?
		time.Sleep(auctionLength)
		delete(p.auctions, a.jobID)
		// a = nil
	}()
	a.mux.Lock()
	defer a.mux.Unlock()
	time.Sleep(auctionLength)

	sqlStmt := `
	SELECT b1.bid_uuid, b1.min_rate
	FROM bids b1
	WHERE b1.job_uuid = $1
		AND b1.late = false
		AND NOT EXISTS(SELECT 1
			FROM bids b2
			INNER JOIN jobs j ON (b2.bid_uuid = j.win_bid_uuid)
			WHERE b2.miner_uuid = b1.miner_uuid
				AND j.active = true
		)
	ORDER BY b1.min_rate
	`
	rows, err := db.Db.Query(sqlStmt, a.jobID)
	if err != nil {
		log.Printf("Error selecting bids from db for job %v: %v\n", a.jobID, err)
		return
	}
	defer check.Err(rows.Close)
	// UPDATE BASED ON LEN(ROWS) AND ORDER BY
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
	sqlStmt = `
	INSERT INTO payments (job_uuid, user_paid, miner_paid)
	VALUES ($1, $2, $3)
	`
	if _, err = db.Db.Exec(sqlStmt, a.jobID, false, false); err != nil {
		log.Printf("Error inserting initial payment into db: %v\n", err)
		return
	}
}

func (a *auction) winner() uuid.UUID {
	a.mux.Lock()
	defer a.mux.Unlock()
	return a.winBid
}
