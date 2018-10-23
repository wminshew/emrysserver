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
var getOutputData app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	jID := vars["jID"]
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		log.Sugar.Errorw("error parsing job ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	p := path.Join("output", jID, "data.tar.gz")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		log.Sugar.Infow("error finding output data.tar.gz on disk",
			"method", r.Method,
			"url", r.URL,
			"jID", jID,
		)
		return getOutputDataCloud(w, r, jUUID, p)
	} else if err != nil {
		log.Sugar.Errorw("error stating output data.tar.gz",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	f, err := os.Open(p)
	if err != nil {
		log.Sugar.Errorw("error opening output data.tar.gz",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	if _, err = io.Copy(w, f); err != nil {
		log.Sugar.Errorw("error copying output data.tar.gz to response",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}

func getOutputDataCloud(w http.ResponseWriter, r *http.Request, jUUID uuid.UUID, p string) *app.Error {
	ctx := r.Context()
	or, err := storage.NewReader(ctx, p)
	if err == storage.ErrObjectNotExist {
		log.Sugar.Errorw("error finding output data.tar.gz in cloud",
			"method", r.Method,
			"url", r.URL,
			"jID", jUUID,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusNoContent, Message: "output data for this job isn't yet available"}
	} else if err != nil {
		log.Sugar.Errorw("error reading from cloud storage",
			"method", r.Method,
			"url", r.URL,
			"jID", jUUID,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if _, err = io.Copy(w, or); err != nil {
		log.Sugar.Errorw("error copying cloud reader to response",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
