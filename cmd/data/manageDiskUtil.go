package main

import (
	"bytes"
	"fmt"
	"github.com/wminshew/emrysserver/pkg/log"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

var (
	pvcPath            = "/data"
	pvcCapGbStr        = os.Getenv("PVC_CAP_GB")
	pvcCapGb           float64
	pvcMaxProjectGbStr = os.Getenv("PVC_MAX_PROJECT_GB")
	pvcMaxProjectGb    float64
	pvcPeriodSecStr    = os.Getenv("PVC_PERIOD_SEC")
	pvcPeriodSec       int
	pvcThresholdStr    = os.Getenv("PVC_THRESHOLD")
	pvcThreshold       float64
)

func initDiskManager() {
	if err := func() error {
		var err error
		if pvcCapGb, err = strconv.ParseFloat(pvcCapGbStr, 64); err != nil {
			return fmt.Errorf("converting PVC_CAP_GB to float64")
		}
		if pvcMaxProjectGb, err = strconv.ParseFloat(pvcMaxProjectGbStr, 64); err != nil {
			return fmt.Errorf("converting PVC_MAX_PROJECT_GB to float64")
		}
		if pvcPeriodSec, err = strconv.Atoi(pvcPeriodSecStr); err != nil {
			return fmt.Errorf("converting PVC_PERIOD_SEC to integer")
		}
		if pvcThreshold, err = strconv.ParseFloat(pvcThresholdStr, 64); err != nil {
			return fmt.Errorf("converting PVC_THRESHOLD to float64")
		}
		return nil
	}(); err != nil {
		panic(err)
	}
}

func checkAndEvictProjects() error {
	var diskSizeGb float64
	var err error
	for diskSizeGb, err = getDirSizeGb(pvcPath); err != nil && diskSizeGb > pvcThreshold*pvcCapGb; diskSizeGb, err = getDirSizeGb(pvcPath) {
		log.Sugar.Infof("Disk size: %.1f / %.1f exceeds threshold %.1f, evicting...", diskSizeGb, pvcCapGb, pvcThreshold*pvcCapGb)
		if err := evictLRUProjectFromDisk(); err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	log.Sugar.Infof("Disk size: %.1f / %.1f", diskSizeGb, pvcCapGb)
	return nil
}

func getDirSizeGb(dir string) (float64, error) {
	var dirSize float64
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			dirSize += float64(info.Size())
		}
		return nil
	}); err != nil {
		return 0, err
	}
	dirSizeGb := dirSize / 1024.0 / 1024.0 / 1024.0
	return dirSizeGb, nil
}

func evictLRUProjectFromDisk() error {
	var lruProject string
	lruTime := time.Now()
	users, err := users()
	if err != nil {
		return err
	}
	for _, user := range users {
		projects, err := projects(user)
		if err != nil {
			return err
		}
		for _, project := range projects {
			projectMd := filepath.Join(project, metadataExt)
			projectStat, err := os.Stat(projectMd)
			if err != nil {
				return err
			}
			projectTime := projectStat.ModTime()
			if projectTime.Before(lruTime) {
				lruProject = project
				lruTime = projectTime
			}
		}
	}
	return os.RemoveAll(lruProject)
}

func users() ([]string, error) {
	files, err := ioutil.ReadDir(pvcPath)
	if err != nil {
		return []string{}, err
	}
	users := []string{}
	for _, f := range files {
		path := filepath.Join(pvcPath, f.Name())
		users = append(users, path)
	}
	return users, nil
}

func projects(userPath string) ([]string, error) {
	files, err := ioutil.ReadDir(userPath)
	if err != nil {
		return []string{}, err
	}
	projects := []string{}
	for _, f := range files {
		path := filepath.Join(userPath, f.Name())
		projects = append(projects, path)
	}
	return projects, nil
}

func touchProjectMd(projectDir string) {
	cmdStr := "touch"
	projectMdPath := filepath.Join(projectDir, metadataExt)
	cmd := exec.Command(cmdStr, projectMdPath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Sugar.Errorf("Error touching %s: %s: %s", projectMdPath, err, stderr.String())
	}
}
