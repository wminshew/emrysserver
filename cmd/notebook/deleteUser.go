package main

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	// "github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"os/exec"
)

// deleteUser handles user removal for notebook service
var deleteUser app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
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

	if err := delUser(jUUID); err != nil {
		log.Sugar.Errorw("error removing notebook user",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error removing notebook user"}
	}

	// if err := db.SetStatusNotebookUserRemoved(jUUID); err != nil {
	// 	log.Sugar.Errorw("error setting notebook user removed",
	// 		"method", r.Method,
	// 		"url", r.URL,
	// 		"err", err.Error(),
	// 	)
	// 	return &app.Error{Code: http.StatusInternalServerError, Message: "error setting notebook user removed"}
	// }
	//
	return nil
}

func delUser(jUUID uuid.UUID) error {
	cmdStr := "deluser"
	args := []string{"--remove-home", jUUID.String()}
	cmd := exec.Command(cmdStr, args...)
	return cmd.Run()
}
