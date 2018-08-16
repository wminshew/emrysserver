package main

import (
	"github.com/gorilla/mux"
	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
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
		// TODO: check if datadir exists, if not, pipe from gcs [and cache locally..?]

		dataDir := filepath.Join("data", uUUID.String(), project, "data")
		if err := archiver.TarGz.Write(w, []string{dataDir}); err != nil {
			log.Sugar.Errorw("failed to get job owner and project",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		return db.SetStatusDataDownloaded(r, jUUID)

		// jobDataDir := filepath.Join("data", "job", jID)
		// var tee io.Reader
		// var dataFile *os.File
		// if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		// 	// if not cached to disk, stream from cloud storage and cache
		// 	ctx := r.Context()
		// 	or, err := bkt.Object(dataPath).NewReader(ctx)
		// 	if err != nil {
		// 		log.Sugar.Errorw("failed to open cloud storage reader",
		// 			"url", r.URL,
		// 			"path", dataPath,
		// 			"err", err.Error(),
		// 			"jID", jID,
		// 		)
		// 		if err := db.SetJobInactive(r, jUUID); err != nil {
		// 			log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
		// 		}
		// 		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		// 	}
		// 	if dataFile, err = os.Create(dataPath); err != nil {
		// 		log.Sugar.Errorw("failed to create disk cache",
		// 			"url", r.URL,
		// 			"path", dataPath,
		// 			"err", err.Error(),
		// 			"jID", jID,
		// 		)
		// 		if err := db.SetJobInactive(r, jUUID); err != nil {
		// 			log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
		// 		}
		// 		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		// 	}
		// 	tee = io.TeeReader(or, dataFile)
		// } else {
		// 	if dataFile, err = os.Open(dataPath); err != nil {
		// 		log.Sugar.Errorw("failed to open data file",
		// 			"url", r.URL,
		// 			"path", dataPath,
		// 			"err", err.Error(),
		// 			"jID", jID,
		// 		)
		// 		if err := db.SetJobInactive(r, jUUID); err != nil {
		// 			log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
		// 		}
		// 		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		// 	}
		// 	tee = io.TeeReader(dataFile, ioutil.Discard)
		// }
		// defer app.CheckErr(r, dataFile.Close)
		//
		// if _, err := io.Copy(w, tee); err != nil {
		// 	log.Sugar.Errorw("failed to copy data file to response writer",
		// 		"url", r.URL,
		// 		"err", err.Error(),
		// 		"jID", jID,
		// 	)
		// 	if err := db.SetJobInactive(r, jUUID); err != nil {
		// 		log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
		// 	}
		// 	return &app.Error{Code: http.StatusInternalServerError, Message: "Internal error"}
		// }
	}
}
