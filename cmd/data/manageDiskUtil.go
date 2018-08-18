package main

import (
	"github.com/wminshew/emrysserver/pkg/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var (
	pvcPath         = "/data"
	pvcCapGbStr     = os.Getenv("PVC_CAP_GB")
	pvcPeriodSecStr = os.Getenv("PVC_PERIOD_SEC")
	pvcThresholdStr = os.Getenv("PVC_THRESHOLD")
)

func runDiskManager() error {
	pvcCapGb, err := strconv.ParseFloat(pvcCapGbStr, 64)
	if err != nil {
		log.Sugar.Errorf("Error converting PVC_CAP_GB to float64")
		return err
	}
	pvcPeriodSec, err := strconv.Atoi(pvcPeriodSecStr)
	if err != nil {
		log.Sugar.Errorf("Error converting PVC_PERIOD_SEC to integer")
		return err
	}
	pvcThreshold, err := strconv.ParseFloat(pvcThresholdStr, 64)
	if err != nil {
		log.Sugar.Errorf("Error converting PVC_THRESHOLD to float64")
		return err
	}
	for {
		if err := checkAndEvict(pvcCapGb, pvcThreshold); err != nil {
			return err
		}
		time.Sleep(time.Duration(pvcPeriodSec) * time.Second)
	}
}

func checkAndEvict(pvcCapGb float64, pvcThreshold float64) error {
	var diskSizeGb float64
	var err error
	for diskSizeGb, err = getDiskSizeGb(pvcPath); err != nil && diskSizeGb > pvcThreshold*pvcCapGb; diskSizeGb, err = getDiskSizeGb(pvcPath) {
		log.Sugar.Infof("Disk size: %.1f, evicting...", diskSizeGb)
		if err := evictLRUProjectFromDisk(); err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	log.Sugar.Infof("Disk size: %.1f", diskSizeGb)
	return nil
}

func getDiskSizeGb(dir string) (float64, error) {
	var diskSize float64
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			diskSize += float64(info.Size())
		}
		return nil
	}); err != nil {
		return 0, err
	}
	diskSizeGb := diskSize / 1024.0 / 1024.0 / 1024.0
	return diskSizeGb, nil
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
