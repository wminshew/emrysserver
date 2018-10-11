package main

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// postAuction creates and runs an auction for job jID
func postAuction() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
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
			return err // already logged
		} else if t != time.Time{} {
			log.Sugar.Infow("user tried to re-auction job",
				"method", r.Method,
				"url", r.URL,
				"jID", jID,
			)
			return nil
		}

		if tDataSynced, tImageBuilt, err := db.GetStatusAuctionPrereqs(r, jUUID); err != nil {
			return err // already logged
		} else if  tDataSynced == time.Time{} || tImageBuilt == time.Time{} {
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

		a = &auction{
			jobID:  jUUID,
			late:   late{bool: false},
			winner: winner{},
		}
		return a.run(r)
	}
}
