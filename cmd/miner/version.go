package main

import (
	"encoding/json"
	"github.com/blang/semver"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/pkg/app"
	"net/http"
)

var latestMinerVer = semver.Version{
	Major: 0,
	Minor: 1,
	Patch: 0,
}

// getVersion returns the latest miner version released
func getVersion() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		resp := creds.VersionResp{
			Version: latestMinerVer.String(),
		}
		err := json.NewEncoder(w).Encode(&resp)
		if err != nil {
			app.Sugar.Errorw("failed to encode miner semver",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		return nil
	}
}
