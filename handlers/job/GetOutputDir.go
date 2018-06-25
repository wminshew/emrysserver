package job

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/flushwriter"
	"io"
	"net/http"
)

// GetOutputDir streams the miner's container execution to the user
func GetOutputDir(w http.ResponseWriter, r *http.Request) *app.Error {
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

	pipe := getDirPipe(jUUID)
	defer deleteDirPipe(jUUID)

	fw := flushwriter.New(w)
	pr := pipe.pr
	if _, err = io.Copy(fw, pr); err != nil {
		app.Sugar.Errorw("failed to copy pipe reader to flushwriter",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "Internal error"}
	}

	return nil
}
