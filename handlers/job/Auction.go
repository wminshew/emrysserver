package job

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrysserver/db"
	"log"
	"math"
	"sync"
	"time"
)

var auctions map[uuid.UUID]*auction = make(map[uuid.UUID]*auction)

type auction struct {
	jobID uuid.UUID
	winner
	late
}

type late struct {
	late bool
	mux  sync.Mutex
}

type winner struct {
	bid uuid.UUID
	mux sync.Mutex
}

const (
	// Buffer after setting bids to late to make sure all in-process bids are added to db
	Buffer = 500 * time.Millisecond
	// Duration of auction
	Duration    = 5 * time.Second
	deleteAfter = 2 * (Duration + Buffer)
)

// NewAuction initializes and runs a new auction
func NewAuction(jID uuid.UUID) {
	a := &auction{
		jobID:  jID,
		late:   late{late: false},
		winner: winner{},
	}
	a.run()
}

func (a *auction) run() {
	a.winner.mux.Lock()
	defer a.winner.mux.Unlock()

	auctions[a.jobID] = a
	defer func() {
		time.Sleep(deleteAfter)
		delete(auctions, a.jobID)
	}()

	time.Sleep(Duration)
	a.late.mux.Lock()
	a.late.late = true
	a.late.mux.Unlock()
	time.Sleep(Buffer)

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
	LIMIT 2
	`
	rows, err := db.Db.Query(sqlStmt, a.jobID)
	if err != nil {
		log.Printf("Error selecting bids for job %v: %v\n", a.jobID, err)
		return
	}
	defer check.Err(rows.Close)

	n := 0
	winRate := math.Inf(1)
	payRate := math.Inf(1)
	for rows.Next() {
		var bidUUID uuid.UUID
		var bidRate float64
		n++
		if err = rows.Scan(&bidUUID, &bidRate); err != nil {
			log.Printf("Error scanning bids: %v\n", err)
			return
		}
		if bidRate < winRate {
			a.winner.bid = bidUUID
			payRate = winRate
			winRate = bidRate
		} else if bidRate < payRate {
			payRate = bidRate
		}
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error scanning bid rows for job %v: %v\n", a.jobID, err)
		return
	}
	if n == 0 {
		log.Printf("No bids received.\n")
		return
	} else if n == 1 {
		payRate = winRate
	}

	log.Printf("%d bid(s) received.\n", n)
	log.Printf("Winning bid: %v\n", a.winner.bid)
	log.Printf("Pay Rate: %v\n", payRate)

	go func() {
		sqlStmt := `
		UPDATE jobs
		SET (win_bid_uuid, pay_rate) = ($1, $2)
		WHERE job_uuid = $3
		`
		_, err = db.Db.Exec(sqlStmt, a.winner.bid, payRate, a.jobID)
		if err != nil {
			log.Printf("Error inserting winning bid info into jobs table: %v\n", err)
			return
		}
		sqlStmt = `
		UPDATE statuses
		SET (auction_completed) = ($1)
		WHERE job_uuid = $2
		`
		_, err = db.Db.Exec(sqlStmt, true, a.jobID)
		if err != nil {
			log.Printf("Error updating job status (auction_completed): %v\n", err)
			return
		}
	}()
}

func (a *auction) winBid() uuid.UUID {
	a.winner.mux.Lock()
	defer a.winner.mux.Unlock()
	return a.winner.bid
}

func (a *auction) lateBid() bool {
	a.late.mux.Lock()
	defer a.late.mux.Unlock()
	return a.late.late
}
