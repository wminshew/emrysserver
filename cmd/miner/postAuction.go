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
