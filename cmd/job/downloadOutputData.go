package main

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"github.com/wminshew/emrysserver/pkg/storage"
	"io"
	"net/http"
	"os"
	"path"
)

// downloadOutputLog downloads the miner's container execution log
func downloadOutputLog() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		jID := vars["jID"]
		jUUID, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("error parsing job ID",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}

		p := path.Join("output", jID, "log")
		if _, err := os.Stat(p); os.IsNotExist(err) {
			log.Sugar.Infow("error finding output log on disk",
				"url", r.URL,
				"jID", jID,
			)
			return getOutputLogCloud(w, r, jUUID, p)
		} else if err != nil {
			log.Sugar.Errorw("error stating output log",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		f, err := os.Open(p)
		if err != nil {
			log.Sugar.Errorw("error opening output log",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		if _, err = io.Copy(w, f); err != nil {
			log.Sugar.Errorw("error copying output log to response",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		return nil
	}
}

func getOutputLogCloud(w http.ResponseWriter, r *http.Request, jUUID uuid.UUID, p string) *app.Error {
	ctx := r.Context()
	or, err := storage.NewReader(ctx, p)
	if err == storage.ErrObjectNotExist {
		log.Sugar.Errorw("error finding output log in cloud",
			"url", r.URL,
			"jID", jUUID,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusNoContent, Message: "output log for this job isn't yet available"}
	} else if err != nil {
		log.Sugar.Errorw("error reading from cloud storage",
			"url", r.URL,
			"jID", jUUID,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if _, err = io.Copy(w, or); err != nil {
		log.Sugar.Errorw("error copying cloud reader to response",
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
