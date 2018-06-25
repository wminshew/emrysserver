package job

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/pkg/app"
	"net/http"
)

// PostBid accepts a job.Bid from miner and adds it to the bids table
func PostBid(w http.ResponseWriter, r *http.Request) *app.Error {
	b := &job.Bid{}
	err := json.NewDecoder(r.Body).Decode(b)
	if err != nil {
		app.Sugar.Errorw("failed to decode json bid body",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "Error parsing request body (json)"}
	}
	b.ID = uuid.NewV4()

	vals := r.URL.Query()
	mIDs, ok := vals["mID"]
	if !ok {
		app.Sugar.Errorw("failed to find mID query",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "Error finding miner ID in jwt claims. Please login again"}
	}
	mID := mIDs[0]

	vars := mux.Vars(r)
	jID := vars["jID"]
	b.JobID, err = uuid.FromString(jID)
	if err != nil {
		app.Sugar.Errorw("failed to parse job ID",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "Error parsing job ID"}
	}

	b.MinerID, err = uuid.FromString(mID)
	if err != nil {
		app.Sugar.Errorw("failed to parse miner ID",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "Error parsing miner ID. Please login again"}
	}

	a, ok := auctions[b.JobID]
	if !ok {
		b.Late = false
	} else {
		b.Late = a.lateBid()
	}
	sqlStmt := `
	INSERT INTO bids (bid_uuid, job_uuid, miner_uuid, min_rate, late)
	VALUES ($1, $2, $3, $4, $5)
	`
	_, err = db.Db.Exec(sqlStmt, b.ID, b.JobID, b.MinerID, b.MinRate, b.Late)
	if err != nil {
		app.Sugar.Errorw("failed to insert bid",
			"url", r.URL,
			"err", err.Error(),
			"bid", b.ID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "Internal error"}
	}
	app.Sugar.Infof("Bid: %+v", b)

	if b.Late {
		app.Sugar.Infof("Late Bid: %v", b.ID)
		return &app.Error{Code: http.StatusOK, Message: "Your bid was late"}
	}

	winbid := a.winBid()
	if !uuid.Equal(winbid, b.ID) {
		return &app.Error{Code: http.StatusOK, Message: "Your bid was not selected"}
	}

	w.Header().Set("Winner", "True")
	return nil
}
