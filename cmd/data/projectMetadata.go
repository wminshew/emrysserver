package main

import (
	"encoding/json"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/pkg/app"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
)

var (
	mdSync   map[string]map[string]*job.FileMetadata
	diskSync map[string]*sync.Mutex
)

func initMetadataSync() {
	mdSync = make(map[string]map[string]*job.FileMetadata)
	diskSync = make(map[string]*sync.Mutex)
}

func getProjectMetadata(r *http.Request, uID, project string, md *map[string]*job.FileMetadata) error {
	uIDProject := path.Join(uID, project)
	if _, ok := diskSync[uIDProject]; !ok {
		diskSync[uIDProject] = &sync.Mutex{}
	}
	diskSync[uIDProject].Lock()
	p := filepath.Join("data", uID, project, ".data_sync_metadata")
	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		diskSync[uIDProject].Unlock()
		return err
	}
	if err := json.NewDecoder(f).Decode(md); err != nil && err != io.EOF {
		diskSync[uIDProject].Unlock()
		return err
	}
	return nil
}

func storeProjectMetadata(r *http.Request, uID, project string, md *map[string]*job.FileMetadata) error {
	uIDProject := path.Join(uID, project)
	defer diskSync[uIDProject].Unlock()
	p := path.Join("data", uID, project, ".data_sync_metadata")
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer app.CheckErr(r, f.Close)
	if err := json.NewEncoder(f).Encode(*md); err != nil {
		return err
	}
	return nil
}

func updateProjectMetadata(r *http.Request, uID, project, relPath string, fileMd *job.FileMetadata) error {
	tempMd := map[string]*job.FileMetadata{}
	if err := getProjectMetadata(r, uID, project, &tempMd); err != nil {
		return err
	}
	tempMd[relPath] = fileMd
	if err := storeProjectMetadata(r, uID, project, &tempMd); err != nil {
		return err
	}
	return nil
}
