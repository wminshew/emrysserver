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

// getOutputData streams the miner's container execution to the user
func getOutputData() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		jID := vars["jID"]
		jUUID, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("failed to parse job ID",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}

		p := path.Join("output", jID, "data.tar.gz")
		if _, err := os.Stat(p); os.IsNotExist(err) {
			log.Sugar.Infow("failed to find output data.tar.gz on disk",
				"url", r.URL,
				"jID", jID,
			)
			return getOutputDataCloud(w, r, jUUID, p)
		} else if err != nil {
			log.Sugar.Errorw("failed to stat output data.tar.gz",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		f, err := os.Open(p)
		if err != nil {
			log.Sugar.Errorw("failed to open output data.tar.gz",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		if _, err = io.Copy(w, f); err != nil {
			log.Sugar.Errorw("failed to copy output data.tar.gz to response",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		return nil
	}
}

func getOutputDataCloud(w http.ResponseWriter, r *http.Request, jUUID uuid.UUID, p string) *app.Error {
	ctx := r.Context()
	or, err := storage.NewReader(ctx, p)
	if err == storage.ErrObjectNotExist {
		log.Sugar.Errorw("failed to find output data.tar.gz in cloud",
			"url", r.URL,
			"jID", jUUID,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusNoContent, Message: "output data for this job isn't yet available"}
	} else if err != nil {
		log.Sugar.Errorw("failed to read from cloud storage",
			"url", r.URL,
			"jID", jUUID,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if _, err = io.Copy(w, or); err != nil {
		log.Sugar.Errorw("failed to copy cloud reader to response",
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
