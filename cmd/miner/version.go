package main

import (
	"encoding/json"
	"github.com/blang/semver"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"os"
)

var latestMinerVer = semver.MustParse(os.Getenv("MINER_SEMVER"))

// getVersion returns the latest miner version released
var getVersion app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	resp := creds.VersionResp{
		Version: latestMinerVer.String(),
	}
	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		log.Sugar.Errorw("error encoding miner semver",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	return nil
}
