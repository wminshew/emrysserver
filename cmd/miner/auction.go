package main

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"math"
	"net/http"
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
	bool
	sync.Mutex
}

type winner struct {
	bid uuid.UUID
	sync.Mutex
}

const (
	buffer      = 500 * time.Millisecond
	duration    = 3 * time.Second
	deleteAfter = duration + buffer
)

func (a *auction) run(r *http.Request) *app.Error {
	a.winner.Lock()
	j := &job.Job{
		ID: a.jobID,
	}
	jMsg := job.Message{
		Message: "New job posted!",
		Job:     j,
	}
	if err := minerManager.Publish("jobs", jMsg); err != nil {
		log.Sugar.Errorw("error publishing job",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", a.jobID,
		)
		a.winner.Unlock()
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	auctions[a.jobID] = a
	success := false
	defer func() {
		a.winner.Unlock()
		if success { // if auction fails, delete immediately to start a new one
			time.Sleep(deleteAfter)
		}
		delete(auctions, a.jobID)
	}()

	time.Sleep(duration)
	a.late.Lock()
	a.late.bool = true
	a.late.Unlock()
	time.Sleep(buffer)

	rows, err := db.GetValidBids(r, a.jobID)
	if err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
	}
	defer app.CheckErr(r, rows.Close)

	n := 0
	winRate := math.Inf(1)
	payRate := math.Inf(1)
	for rows.Next() {
		var bidUUID uuid.UUID
		var bidRate float64
		n++
		if err = rows.Scan(&bidUUID, &bidRate); err != nil {
			log.Sugar.Errorw("error scanning bids",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", a.jobID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
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
		log.Sugar.Errorw("error scanning bids",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", a.jobID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	if n == 0 {
		log.Sugar.Infof("no bids received")
		return &app.Error{Code: http.StatusPaymentRequired, Message: "no bids received, please try again"}
	} else if n == 1 {
		payRate = winRate
	}
	success = true
	log.Sugar.Infow("successful auction",
		"bids", n,
		"winner", a.winner.bid,
		"rate", payRate,
	)
	go monitorJob(a.jobID)

	return db.SetJobWinnerAndAuctionStatus(r, a.jobID, a.winner.bid, payRate)
}

func (a *auction) winBid() uuid.UUID {
	a.winner.Lock()
	defer a.winner.Unlock()
	return a.winner.bid
}

func (a *auction) lateBid() bool {
	a.late.Lock()
	defer a.late.Unlock()
	return a.late.bool
}
