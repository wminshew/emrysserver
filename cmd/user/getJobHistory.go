package main

import (
	"encoding/json"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// getJobHistory returns the accounts job history
var getJobHistory app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	aID := r.Header.Get("X-Jwt-Claims-Subject")
	aUUID, err := uuid.FromString(aID)
	if err != nil {
		log.Sugar.Errorw("error parsing account ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	// TODO: add job type, etc
	// jobHistory := make(map[uuid.UUID]job)
	jobHistory := []uuid.UUID{}

	// TODO: break into user / miner job histories?
	rows, err := db.GetAccountJobHistory(aUUID)
	if err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Sugar.Errorf("Error closing rows")
		}
	}()

	for rows.Next() {
		var aUUID uuid.UUID
		if err = rows.Scan(&aUUID); err != nil {
			log.Sugar.Errorw("error scanning job history",
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		jobHistory = append(jobHistory, aUUID)
	}
	if err = rows.Err(); err != nil {
		log.Sugar.Errorw("error scanning job history",
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if err := json.NewEncoder(w).Encode(&jobHistory); err != nil {
		log.Sugar.Errorw("error encoding account job history",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
