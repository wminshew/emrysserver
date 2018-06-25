package job

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/pkg/app"
	"math"
	"sync"
	"time"
)

var auctions = make(map[uuid.UUID]*auction)

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
	buffer      = 500 * time.Millisecond
	duration    = 5 * time.Second
	deleteAfter = 2 * (duration + buffer)
)

func newAuction(jID uuid.UUID) {
	a := &auction{
		jobID:  jID,
		late:   late{late: false},
		winner: winner{},
	}
	go a.run()
}

func (a *auction) run() {
	a.winner.mux.Lock()
	auctions[a.jobID] = a
	defer func() {
		a.winner.mux.Unlock()
		time.Sleep(deleteAfter)
		delete(auctions, a.jobID)
	}()

	time.Sleep(duration)
	a.late.mux.Lock()
	a.late.late = true
	a.late.mux.Unlock()
	time.Sleep(buffer)

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
		app.Sugar.Errorw("failed to query bids",
			// "url", r.URL,
			"err", err.Error(),
			"jID", a.jobID,
		)
		// return &app.Error{http.StatusInternalServerError, "Internal error"}
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
			app.Sugar.Errorw("failed to scan bids",
				// "url", r.URL,
				"err", err.Error(),
				"jID", a.jobID,
			)
			// return &app.Error{http.StatusInternalServerError, "Internal error"}
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
		app.Sugar.Errorw("failed to scan bids",
			// "url", r.URL,
			"err", err.Error(),
			"jID", a.jobID,
		)
		// return &app.Error{http.StatusInternalServerError, "Internal error"}
		return
	}
	if n == 0 {
		app.Sugar.Infof("No bids received")
		return
	} else if n == 1 {
		payRate = winRate
	}

	app.Sugar.Infof("%d bid(s) received", n)
	app.Sugar.Infof("Winning bid: %v", a.winner.bid)
	app.Sugar.Infof("Pay Rate: %v", payRate)

	sqlStmt = `
	UPDATE jobs
	SET (win_bid_uuid, pay_rate) = ($1, $2)
	WHERE job_uuid = $3
	`
	if _, err = db.Db.Exec(sqlStmt, a.winner.bid, payRate, a.jobID); err != nil {
		app.Sugar.Errorw("failed to update job",
			// "url", r.URL,
			"err", err.Error(),
			"jID", a.jobID,
		)
		// return &app.Error{http.StatusInternalServerError, "Internal error"}
		return
	}
	sqlStmt = `
	UPDATE statuses
	SET (auction_completed) = ($1)
	WHERE job_uuid = $2
	`
	if _, err = db.Db.Exec(sqlStmt, true, a.jobID); err != nil {
		app.Sugar.Errorw("failed to update job status",
			// "url", r.URL,
			"err", err.Error(),
			"jID", a.jobID,
		)
		// return &app.Error{http.StatusInternalServerError, "Internal error"}
		return
	}
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
