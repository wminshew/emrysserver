package main

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// verifyUser handles user verification for notebook service
var verifyUser app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	jID := r.URL.Query().Get("jID")
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		log.Sugar.Errorw("error parsing job ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	// TODO: verify should return user & miner keys, right?
	sshKey, err := db.GetJobSSHKeyUser(jUUID)
	if err != nil {
		log.Sugar.Errorw("error getting job ssh pubkey for user",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error getting job ssh pubkey for user"}
	}

	// TODO: remove log
	log.Sugar.Infof("%s", sshKey)

	if _, err := w.Write([]byte(sshKey)); err != nil {
		log.Sugar.Errorw("error writing job ssh key",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error writing job ssh key"}
	}

	return nil
}
