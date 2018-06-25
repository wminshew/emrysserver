package miner

import (
	"compress/zlib"
	"docker.io/go-docker"
	"github.com/gorilla/mux"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/check"
	"io"
	"net/http"
)

// Image sends the {jID} job docker image to the miner for execution
func Image(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	jID := vars["jID"]
	mID := vars["mID"]

	ctx := r.Context()
	cli, err := docker.NewEnvClient()
	if err != nil {
		app.Sugar.Errorw("failed to create docker client",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
			"mID", mID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	img, err := cli.ImageSave(ctx, []string{jID})
	defer check.Err(r, img.Close)

	zw := zlib.NewWriter(w)
	defer check.Err(r, zw.Close)

	if _, err = io.Copy(zw, img); err != nil {
		app.Sugar.Errorw("failed to copy image to zlib response writer",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
			"mID", mID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	sqlStmt := `
	UPDATE statuses
	SET (image_downloaded) = ($1)
	WHERE job_uuid = $2
	`
	if _, err = db.Db.Exec(sqlStmt, true, jID); err != nil {
		app.Sugar.Errorw("failed to update job status",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
			"mID", mID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
