package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// postBid accepts a job.Bid from a miner
var postBid app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	var err error
	b := &job.Bid{}
	if err = json.NewDecoder(r.Body).Decode(b); err != nil {
		log.Sugar.Errorw("error decoding json bid body",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json bid request body"}
	}
	b.ID = uuid.NewV4()

	vars := mux.Vars(r)
	jID := vars["jID"]
	if b.JobID, err = uuid.FromString(jID); err != nil {
		log.Sugar.Errorw("error parsing job ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	mID := r.Header.Get("X-Jwt-Claims-Subject")
	if b.MinerID, err = uuid.FromString(mID); err != nil {
		log.Sugar.Errorw("error parsing miner ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", b.JobID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing miner ID"}
	}

	a, ok := auctions[b.JobID]
	if !ok {
		b.Late = true
		log.Sugar.Infof("Late bid: %+v", b)
		return &app.Error{Code: http.StatusBadRequest, Message: "your bid was late"}
	}
	b.Late = a.lateBid()

	if b.Specs.Rate <= 0 {
		log.Sugar.Errorw("non-postitive bid rate",
			"method", r.Method,
			"url", r.URL,
			"jID", jID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "must submit positive bid rate"}
	}
	if b.Specs.GPU, ok = job.ValidateGPU(b.Specs.GPU); !ok {
		log.Sugar.Errorw("invalid gpu",
			"method", r.Method,
			"url", r.URL,
			"jID", jID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "invalid gpu"}
	}
	if b.Specs.RAM == 0 {
		log.Sugar.Errorw("no ram spec in bid",
			"method", r.Method,
			"url", r.URL,
			"jID", jID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "must submit ram allocation with bid"}
	}
	if b.Specs.Disk == 0 {
		log.Sugar.Errorw("no disk spec in bid",
			"method", r.Method,
			"url", r.URL,
			"jID", jID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "must submit disk allocation with bid"}
	}
	if b.Specs.Pcie != 16 && b.Specs.Pcie != 8 && b.Specs.Pcie != 4 && b.Specs.Pcie != 2 && b.Specs.Pcie != 1 {
		log.Sugar.Errorw("invalid pcie in bid",
			"method", r.Method,
			"url", r.URL,
			"jID", jID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "invalid pcie"}
	}

	meetsGPUReq, err := job.CompareGPU(b.Specs.GPU, a.requirements.GPU)
	if err != nil {
		log.Sugar.Errorw("error comparing gpus",
			"method", r.Method,
			"url", r.URL,
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	meetsReqs := b.Specs.Rate <= a.requirements.Rate &&
		meetsGPUReq &&
		b.Specs.RAM >= a.requirements.RAM &&
		b.Specs.Disk >= a.requirements.Disk &&
		b.Specs.Pcie >= a.requirements.Pcie

	log.Sugar.Infof("%+v meets reqs: %v", b, meetsReqs) // TODO: remove

	if err := db.InsertBid(r, b, meetsReqs); err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	log.Sugar.Infof("Bid %s (rate: %.2f, late: %s, meets reqs: %s) for job %s received!", b.ID.String(), b.Specs.Rate, b.Late, meetsReqs, b.JobID.String())

	if b.Late {
		return &app.Error{Code: http.StatusBadRequest, Message: "your bid was late"}
	}

	winbid := a.winBid()
	if !uuid.Equal(winbid, b.ID) {
		return &app.Error{Code: http.StatusPaymentRequired, Message: "your bid was not selected"}
	}

	return nil
}
