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

// postBid accepts a job.Bid from miner and calls handlers/job.PostBid
func postBid() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		var err error
		vars := mux.Vars(r)
		b := &job.Bid{}
		if err = json.NewDecoder(r.Body).Decode(b); err != nil {
			log.Sugar.Errorw("failed to decode json bid body",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json bid request body"}
		}
		b.ID = uuid.NewV4()

		jID := vars["jID"]
		if b.JobID, err = uuid.FromString(jID); err != nil {
			log.Sugar.Errorw("failed to parse job ID",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}

		mID := vars["mID"]
		if b.MinerID, err = uuid.FromString(mID); err != nil {
			log.Sugar.Errorw("failed to parse miner ID",
				"url", r.URL,
				"err", err.Error(),
				"jID", b.JobID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing miner ID. Please login again"}
		}

		a, ok := auctions[b.JobID]
		if !ok {
			b.Late = false
		} else {
			b.Late = a.lateBid()
		}

		if err := db.InsertBid(r, b); err != nil {
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		log.Sugar.Infof("Bid %s (rate: %.2f, late: %s) for job %s received!", b.ID.String(), b.MinRate, b.Late, b.JobID.String())

		if b.Late {
			return &app.Error{Code: http.StatusBadRequest, Message: "your bid was late"}
		}

		winbid := a.winBid()
		if !uuid.Equal(winbid, b.ID) {
			return &app.Error{Code: http.StatusPaymentRequired, Message: "your bid was not selected"}
		}

		return nil
	}
}
