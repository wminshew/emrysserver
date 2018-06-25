package job

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/pkg/app"
	"net/http"
	"time"
)

// GetAuctionSuccess returns whether an auction is successful
func GetAuctionSuccess(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	jID := vars["jID"]
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		app.Sugar.Errorw("failed to parse job ID",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "Error parsing job ID"}
	}

	time.Sleep(duration)

	a, ok := auctions[jUUID]
	if ok {
		winBid := a.winBid()
		if uuid.Equal(winBid, uuid.Nil) {
			// TODO: once cloud services attached, there should always be at least one bid
			return &app.Error{Code: http.StatusPaymentRequired, Message: "no bids received"}
		}

		return nil
	}

	var success bool
	sqlStmt := `
	SELECT auction_completed
	FROM statuses
	WHERE job_uuid = $1
	`
	if err = db.Db.QueryRow(sqlStmt, jID).Scan(&success); err != nil {
		app.Sugar.Errorw("error querying status",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	if success {
		return nil
	}

	return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
}
