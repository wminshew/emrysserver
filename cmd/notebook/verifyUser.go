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

	log.Sugar.Infof("VERIFYING USER: %s", jID) // TODO: remove

	userSSHKeyPub, err := db.GetJobSSHKeyPubUser(jUUID)
	if err != nil {
		log.Sugar.Errorw("error getting job ssh pubkey for user",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error getting job ssh pubkey for user"}
	}

	log.Sugar.Infof("USER KEY: %s", userSSHKeyPub) // TODO: remove

	minerSSHKeyPub, err := db.GetJobSSHKeyPubMiner(jUUID)
	if err != nil {
		log.Sugar.Errorw("error getting job ssh pubkey for miner",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error getting job ssh pubkey for miner"}
	}

	log.Sugar.Infof("MINER KEY: %s", minerSSHKeyPub) // TODO: remove

	sshKeyFile := append([]byte(userSSHKeyPub), append([]byte("\n"), []byte(minerSSHKeyPub)...)...)

	log.Sugar.Infof("KEY FILE: %s", string(sshKeyFile))

	if _, err := w.Write(sshKeyFile); err != nil {
		log.Sugar.Errorw("error writing job ssh key file",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error writing job ssh key file"}
	}

	return nil
}
