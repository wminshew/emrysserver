package job

import (
	"cloud.google.com/go/storage"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/flushwriter"
	"io"
	"net/http"
	"path"
)

// GetOutputLog streams the miner's container execution to the user
func GetOutputLog(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	jID := vars["jID"]
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		app.Sugar.Errorw("failed to parse job ID",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	var reader io.Reader
	p := path.Join("job", jID, "output", "log")
	ctx := r.Context()
	reader, err = bkt.Object(p).NewReader(ctx)
	if err == storage.ErrObjectNotExist {
		pr, pw := io.Pipe()
		logPipes[jUUID] = &pipe{
			r: pr,
			w: pw,
		}
		reader = pr
		defer delete(logPipes, jUUID)
	} else if err != nil {
		app.Sugar.Errorw("failed to read from cloud storage",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	fw := flushwriter.New(w)
	if _, err = io.Copy(fw, reader); err != nil {
		app.Sugar.Errorw("failed to copy pipe reader to flushwriter",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
