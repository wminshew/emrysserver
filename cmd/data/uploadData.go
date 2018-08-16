package main

import (
	"compress/zlib"
	"crypto/md5"
	"encoding/base64"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

// uploadData receives the map of the user's data set metadata and determines which files needed to be re-uploaded
func uploadData() app.Handler {
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

		uID := vars["uID"]
		_, err = uuid.FromString(uID)
		if err != nil {
			log.Sugar.Errorw("failed to parse user ID",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing user ID"}
		}

		project := vars["project"]
		projectDir := filepath.Join("data", uID, project)
		// TODO: make sure project dir exists; if it doesn't, download from gcs?
		// TODO: it would be very odd if it didn't, but it seems .. possible; then again, maybe just throw an error so user
		// has to re-submit job
		relPath := vars["relPath"]
		// TODO: more validation on relPath? e.g. avoid '..' and stuff..
		uploadPath := filepath.Join(projectDir, "data", relPath)
		uploadDir := filepath.Dir(uploadPath)

		if _, err = os.Stat(uploadDir); os.IsNotExist(err) {
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				log.Sugar.Errorw("failed to create upload dir",
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
			}
		} else if err != nil {
			log.Sugar.Errorw("failed to get upload dir stat",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		f, err := os.Create(uploadPath)
		if err != nil {
			log.Sugar.Errorw("failed to create file",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		defer app.CheckErr(r, f.Close)

		pr, pw := io.Pipe()
		go func() {
			defer app.CheckErr(r, pw.Close)
			if _, err := io.Copy(pw, r.Body); err != nil {
				log.Sugar.Errorw("failed to copy request body to pipe writer",
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				return
			}
		}()

		zr, err := zlib.NewReader(pr)
		if err != nil {
			log.Sugar.Errorw("failed to create zlib reader",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		h := md5.New()
		tee := io.TeeReader(zr, h)
		if _, err := io.Copy(f, tee); err != nil {
			log.Sugar.Errorw("failed to copy zlib reader to disk",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		app.CheckErr(r, zr.Close)

		hStr := base64.StdEncoding.EncodeToString(h.Sum(nil))
		uIDProject := path.Join(uID, project)
		fileMd := mdSync[uIDProject][relPath]
		if hStr != fileMd.Hash {
			log.Sugar.Errorw("uploaded file checksum doesn't match",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
				"relPath", relPath,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "file upload checksum mismatch"}
		}

		if err := updateProjectMetadata(r, uID, project, relPath, fileMd); err != nil {
			log.Sugar.Errorw("failed to store project metatdata",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		delete(mdSync[uIDProject], relPath)

		if len(mdSync[uIDProject]) == 0 {
			delete(diskSync, uIDProject)
			return db.SetStatusDataSynced(r, jUUID)
		}
		// TODO: upload file, metadata to gcs? [ideally as job not tied to returning here]
		return nil
	}
}
