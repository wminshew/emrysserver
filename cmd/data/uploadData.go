package main

import (
	"compress/zlib"
	"crypto/md5"
	"encoding/base64"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/validate"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
)

// uploadData receives the map of the user's data set metadata and determines which files needed to be re-uploaded
var uploadData app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
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

	uID := r.Header.Get("X-Jwt-Claims-Subject")
	_, err = uuid.FromString(uID)
	if err != nil {
		log.Sugar.Errorw("error parsing user ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing user ID"}
	}

	project := vars["project"]
	projectDir := filepath.Join("data", uID, project)
	if _, err = os.Stat(projectDir); os.IsNotExist(err) {
		log.Sugar.Errorw("project dir doesn't exist on disk",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
			"project", project,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	if projectSize, err := getDirSizeGb(projectDir); err != nil {
		log.Sugar.Errorw("error retrieving size of project dir",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
			"project", project,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	} else if projectSize > pvcMaxProjectGb {
		log.Sugar.Errorw("project is over maximum project size",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
			"project", project,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "project data set is over limit, cannot upload"}
	}

	relPath := vars["relPath"]
	relPathRegexp := validate.RelPathRegexp()
	relPathAntiRegexp := validate.RelPathAntiRegexp()
	if !relPathRegexp.MatchString(relPath) || relPathAntiRegexp.MatchString(relPath) {
		log.Sugar.Errorw("invalid upload path",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
			"relPath", relPath,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "invalid upload path"}
	}

	uploadPath := filepath.Join(projectDir, "data", relPath)
	uploadDir := filepath.Dir(uploadPath)
	if _, ok := diskSync[uploadPath]; !ok {
		diskSync[uploadPath] = &sync.Mutex{}
	}
	diskSync[uploadPath].Lock()
	defer diskSync[uploadPath].Unlock()

	if _, err = os.Stat(uploadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			log.Sugar.Errorw("error creating upload dir",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
	} else if err != nil {
		log.Sugar.Errorw("error retrieving upload dir stat",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	f, err := os.Create(uploadPath)
	if err != nil {
		log.Sugar.Errorw("error creating file",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	defer app.CheckErr(r, f.Close)

	pr, pw := io.Pipe()
	// defer app.CheckErr(r, pr.Close) TODO: maybe should move below to after zr.Close
	defer app.CheckErr(r, pw.Close)
	go func() {
		// defer app.CheckErr(r, pw.Close) TODO: does moving the pw.close to outside the go func remove the error?
		if _, err := io.Copy(pw, r.Body); err != nil && err != io.ErrClosedPipe {
			log.Sugar.Errorw("error copying request body to pipe writer",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return
		}
	}()

	zr, err := zlib.NewReader(pr)
	if err != nil {
		log.Sugar.Errorw("error creating zlib reader",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	h := md5.New()
	tee := io.TeeReader(zr, h)
	if _, err := io.Copy(f, tee); err != nil {
		log.Sugar.Errorw("error copying zlib reader to disk",
			"method", r.Method,
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
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
			"relPath", relPath,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "file upload checksum mismatch"}
	}

	if err := updateProjectMetadata(r, uID, project, relPath, fileMd); err != nil {
		log.Sugar.Errorw("error storing project metatdata",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	delete(mdSync[uIDProject], relPath)

	if len(mdSync[uIDProject]) == 0 {
		delete(diskSync, uIDProject)
		go func() {
			if err := uploadProject(projectDir); err != nil {
				log.Sugar.Errorw("error uploading project dir",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
					"project", project,
				)
				return
			}
		}()
		go func() {
			if err := checkAndEvictProjects(); err != nil {
				log.Sugar.Errorf("Error managing disk utilization: %v\n", err)
				return
			}
		}()
		return db.SetStatusDataSynced(r, jUUID)
	}
	return nil
}
