package main

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

// postMiner creates a new ssh-key-pair for the job-winning miner
var postMiner app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
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

	sshKeyPath := filepath.Join("ssh-keys", fmt.Sprintf("%s-miner", jUUID.String()))
	sshKeyPubPath := sshKeyPath + ".pub"
	if err := createSSHKeyPair(sshKeyPath, sshKeyPubPath); err != nil {
		log.Sugar.Errorw("error creating ssh key pair",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error creating ssh key pair"}
	}
	defer func() {
		if err := deleteSSHKeyPair(sshKeyPath, sshKeyPubPath); err != nil {
			log.Sugar.Errorw("error deleting ssh key pair",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
		}
	}()

	sshKeyPubBytes, err := ioutil.ReadFile(sshKeyPubPath)
	if err != nil {
		log.Sugar.Errorw("error reading ssh pubkey file",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error reading ssh pubkey file"}
	}
	sshKeyPub := string(sshKeyPubBytes)

	if err := db.SetJobSSHKeyPubMiner(jUUID, sshKeyPub); err != nil {
		log.Sugar.Errorw("error setting job ssh pubkey for miner",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error setting job ssh pubkey for miner"}
	}

	sshKeyBytes, err := ioutil.ReadFile(sshKeyPath)
	if err != nil {
		log.Sugar.Errorw("error reading ssh key file",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error reading ssh key file"}
	}
	if _, err := w.Write(sshKeyBytes); err != nil {
		log.Sugar.Errorw("error writing job ssh key to miner",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error writing job ssh key to miner"}
	}

	return nil
}
