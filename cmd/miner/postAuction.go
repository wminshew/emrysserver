package main

import (
	"encoding/json"
	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"math"
	"net/http"
)

const (
	defaultRAM  = 8
	defaultDisk = 25
	defaultPcie = 8
)

// postAuction creates and runs an auction for job jID
var postAuction app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	jID := vars["jID"]
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		log.Sugar.Errorw("error parsing job ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	if t, err := db.GetStatusAuctionCompleted(r, jUUID); err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // err already logged
	} else if !t.IsZero() {
		log.Sugar.Infow("user tried to re-auction job",
			"method", r.Method,
			"url", r.URL,
			"jID", jID,
		)
		return nil
	}

	if tDataSynced, tImageBuilt, err := db.GetStatusAuctionPrereqs(r, jUUID); err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // err already logged
	} else if tDataSynced.IsZero() || tImageBuilt.IsZero() {
		log.Sugar.Infow("user tried to auction job without completing prereqs",
			"method", r.Method,
			"url", r.URL,
			"jID", jID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "must successfully sync data & build image before auctioning job"}
	}

	a, ok := auctions[jUUID]
	if ok {
		winBid := a.winBid()
		if uuid.Equal(winBid, uuid.Nil) {
			return &app.Error{Code: http.StatusPaymentRequired, Message: "no bids received"}
		}
		return nil
	}

	reqs := &job.Specs{}
	if err := json.NewDecoder(r.Body).Decode(reqs); err != nil {
		return &app.Error{Code: http.StatusBadRequest, Message: "error decoding request body"}
	}

	if reqs.Rate == 0 {
		reqs.Rate = math.Inf(0)
	} else if reqs.Rate < 0 {
		log.Sugar.Errorw("negative job rate",
			"method", r.Method,
			"url", r.URL,
			"jID", jID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "negative job rate"}
	}
	if reqs.GPU, ok = job.ValidateGPU(reqs.GPU); !ok {
		log.Sugar.Errorw("invalid gpu",
			"method", r.Method,
			"url", r.URL,
			"jID", jID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "invalid gpu"}
	}
	if reqs.RAM == 0 {
		reqs.RAM = defaultRAM * humanize.GByte
	}
	if reqs.Disk == 0 {
		reqs.Disk = defaultDisk * humanize.GByte
	}
	if reqs.Pcie == 0 {
		reqs.Pcie = defaultPcie
	} else if reqs.Pcie != 16 && reqs.Pcie != 8 && reqs.Pcie != 4 && reqs.Pcie != 2 && reqs.Pcie != 1 {
		log.Sugar.Errorw("invalid pcie",
			"method", r.Method,
			"url", r.URL,
			"jID", jID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "invalid pcie"}
	}

	if err := db.InsertJobSpecs(r, jUUID, reqs); err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // err already logged
	}

	a = &auction{
		jobID:        jUUID,
		late:         late{bool: false},
		winner:       winner{},
		requirements: reqs,
	}
	return a.run(r)
}
