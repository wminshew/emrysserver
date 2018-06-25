package user

import (
	"encoding/json"
	"github.com/blang/semver"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/pkg/app"
	"net/http"
)

var latestUserVer = semver.Version{
	Major: 0,
	Minor: 1,
	Patch: 0,
}

// GetVersion returns the latest user version released
func GetVersion(w http.ResponseWriter, r *http.Request) *app.Error {
	resp := creds.VersionResp{
		Version: latestUserVer.String(),
	}
	err := json.NewEncoder(w).Encode(&resp)
	if err != nil {
		app.Sugar.Errorw("failed to encode user semver",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
