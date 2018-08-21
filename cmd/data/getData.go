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
			log.Sugar.Errorw("failed to parse job ID",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}

		uUUID, project, err := db.GetJobOwnerAndProject(r, jUUID)
		if err != nil {
			log.Sugar.Errorw("failed to get job owner and project",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		projectDir := filepath.Join("data", uUUID.String(), project)
		if _, err = os.Stat(projectDir); os.IsNotExist(err) {
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				log.Sugar.Errorw("failed to get server project metadata",
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
			}
			if err := downloadProject(projectDir); err != nil {
				log.Sugar.Errorw("failed to download project from gcs",
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
			log.Sugar.Errorw("failed to get project directory",
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
		if err := archiver.TarGz.Write(w, []string{dataDir}); err != nil {
			log.Sugar.Errorw("failed to get job owner and project",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		return db.SetStatusDataDownloaded(r, jUUID)
	}
}
