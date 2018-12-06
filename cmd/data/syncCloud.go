package main

import (
	"bytes"
	"fmt"
	"github.com/wminshew/emrysserver/pkg/log"
	"os"
	"os/exec"
	"path/filepath"
)

const bkt = "gs://emrys-dev"

func projectExists(projectDir string) bool {
	// projectDir = /data/{uID}/{project}
	// gsutil -q stat gs://emrys-dev/data/{uID}/{project} to make sure project folder exists first
	// https://cloud.google.com/storage/docs/gsutil/commands/stat
	cmdStr := "gsutil"
	src := fmt.Sprintf("%s/%s", bkt, projectDir)
	args := []string{"-q", "stat", src}
	cmd := exec.Command(cmdStr, args...)
	err := cmd.Run()
	return (err == nil)
}

func downloadProject(projectDir string) error {
	// projectDir = /data/{uID}/{project}
	// gsutil -m cp -r gs://emrys-dev/data/{uID}/{project} /data/{uID}
	// https://cloud.google.com/storage/docs/gsutil/commands/cp
	cmdStr := "gsutil"
	src := fmt.Sprintf("%s/%s", bkt, projectDir)
	dst := filepath.Dir(projectDir)
	args := []string{"-m", "cp", "-r", src, dst}
	cmd := exec.Command(cmdStr, args...)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	cmd.Env = os.Environ()
	err := cmd.Run()
	log.Sugar.Infof(out.String())
	if err != nil {
		log.Sugar.Errorf("%s: %s", err, stderr.String())
	}
	return err
}

func uploadProject(projectDir string) error {
	// projectDir = /data/{uID}/{project}
	// gsutil -m rsync -d -r {projectDir} gs://emrys-dev/{projectDir}
	// https://cloud.google.com/storage/docs/gsutil/commands/rsync
	cmdStr := "gsutil"
	src := projectDir
	dst := fmt.Sprintf("%s/%s", bkt, projectDir)
	args := []string{"-m", "rsync", "-d", "-r", src, dst}
	cmd := exec.Command(cmdStr, args...)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	cmd.Env = os.Environ()
	err := cmd.Run()
	log.Sugar.Infof(out.String())
	if err != nil {
		log.Sugar.Errorf("%s: %s", err, stderr.String())
	}
	return err
}
