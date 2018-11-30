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

var latestUserVer = semver.MustParse(os.Getenv("USER_SEMVER"))

// getVersion returns the latest user version released
var getVersion app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	resp := creds.VersionResp{
		Version: latestUserVer.String(),
	}
	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		log.Sugar.Errorw("error encoding user semver",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	return nil
}
