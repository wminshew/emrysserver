package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	// "github.com/wminshew/emrysserver/pkg/storage"
	"net/http"
)

// getOutputLog streams the miner's container execution to the user
func getOutputLog() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		jID := vars["jID"]
		_, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("error parsing job ID",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}

		q := r.URL.Query()
		q.Set("category", jID)
		q.Set("timeout", fmt.Sprintf("%d", maxTimeout))
		r.URL.RawQuery = q.Encode()
		jobsManager.SubscriptionHandler(w, r)

		return nil
	}
}

// var reader io.Reader
// p := path.Join("job", jID, "output", "log")
// ctx := r.Context()
// reader, err = storage.NewReader(ctx, p)
// if err == storage.ErrObjectNotExist {
//
// } else if err != nil {
// 	log.Sugar.Errorw("error reading from cloud storage",
// 		"url", r.URL,
// 		"err", err.Error(),
// 		"jID", jID,
// 	)
// 	return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
// }
//
// if _, err = io.Copy(w, reader); err != nil {
// 	log.Sugar.Errorw("error copying pipe reader to response writer",
// 		"url", r.URL,
// 		"err", err.Error(),
// 		"jID", jID,
// 	)
// 	return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
// }
