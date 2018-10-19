package main

import (
	"github.com/gorilla/mux"
	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"os"
	"path/filepath"
)

// getData sends data.tar.gz, if it exists, associated with job jID to the miner
func getData() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
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

		if t, err := db.GetStatusDataDownloaded(r, jUUID); err != nil {
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged err
		} else if !t.IsZero() {
			log.Sugar.Infow("miner tried to re-download data",
				"method", r.Method,
				"url", r.URL,
				"jID", jID,
			)
			return nil
		}

		uUUID, project, err := db.GetJobOwnerAndProject(r, jUUID)
		if err != nil {
			log.Sugar.Errorw("error retrieving job owner and project",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		projectDir := filepath.Join("data", uUUID.String(), project)
		if _, err = os.Stat(projectDir); os.IsNotExist(err) {
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				log.Sugar.Errorw("error making project dir",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
			}
			if err := downloadProject(projectDir); err != nil {
				log.Sugar.Errorw("error downloading project from gcs",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
			}
			go func() {
				if err := checkAndEvictProjects(); err != nil {
					log.Sugar.Errorf("Error managing disk utilization: %v\n", err)
					return
				}
			}()
		} else if err != nil {
			log.Sugar.Errorw("error stating project dir",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		go func() {
			touchProjectMd(projectDir)
		}()
		dataDir := filepath.Join(projectDir, "data")
		if _, err = os.Stat(dataDir); !os.IsNotExist(err) {
			if err := archiver.TarGz.Write(w, []string{dataDir}); err != nil {
				log.Sugar.Errorw("error writing tar gzipped data dir",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
			}
		}

		return db.SetStatusDataDownloaded(r, jUUID)
	}
}
