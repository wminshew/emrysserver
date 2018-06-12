package miner

import (
	"encoding/json"
	"github.com/blang/semver"
	"github.com/wminshew/emrys/pkg/creds"
	"log"
	"net/http"
)

var latestMinerVer = semver.Version{
	Major: 0,
	Minor: 1,
	Patch: 0,
}

// GetVersion returns the latest miner version released
func GetVersion(w http.ResponseWriter, r *http.Request) {
	resp := creds.VersionResp{
		Version: latestMinerVer.String(),
	}
	err := json.NewEncoder(w).Encode(&resp)
	if err != nil {
		log.Printf("Error encoding miner semver: %v\n", err)
		http.Error(w, "Internal error!", http.StatusInternalServerError)
		return
	}
}
