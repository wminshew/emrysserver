package handlers

import (
	"fmt"
	"github.com/mholt/archiver"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func JobUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// parse multipart Form request; limit memory usage
		// (residual should end up temporarily on disk)
		maxMemory := int64(1) << 31
		err := r.ParseMultipartForm(maxMemory)
		if err != nil {
			log.Printf("Error parsing request: %v\n", err)
		}

		// if doesn't exist yet, create user directory for uploads
		username, _, _ := r.BasicAuth()
		// TODO: add extra directory layer for job number; return job number to client
		userDir := "./user-upload/" + username + "/"
		// TODO: THIS FEELS DANGEROUS; IS THERE A SAFER WAY?
		// error behavior without adjusting umask:
		// directory without execution / writing bits cannot be written to
		oldUmask := syscall.Umask(022)
		if err = os.MkdirAll(userDir, 0777); err != nil {
			log.Printf("Error creating user directory %s: %v\n", userDir, err)
		}
		_ = syscall.Umask(oldUmask)

		// open reader on Requirements file
		requirementsTempFile, requirementsHandler, err := r.FormFile("Requirements")
		if err != nil {
			log.Printf("Error reading requirements form file: %v\n", err)
		}
		defer requirementsTempFile.Close()

		// create new file to save down Requirements file on disk
		requirementsPath := userDir + filepath.Base(requirementsHandler.Filename)
		// TODO: may have to chmod this file later to execute; may need to update
		// file permissions here for ease later
		requirementsFile, err := os.OpenFile(requirementsPath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Printf("Error opening requirements file: %v\n", err)
		}
		defer requirementsFile.Close()

		// copy Requirements file contents to disk
		n_requirements, err := io.Copy(requirementsFile, requirementsTempFile)
		if err != nil {
			log.Printf("Error copying requirements file to disk: %v\n", err)
		}

		// open reader on Train file
		trainTempFile, trainHandler, err := r.FormFile("Train")
		if err != nil {
			log.Printf("Error reading train form file: %v\n", err)
		}
		defer trainTempFile.Close()

		// create new file to save down Train file on disk
		trainPath := userDir + filepath.Base(trainHandler.Filename)
		// TODO: do i need to use Umask? running with python, still not sure if safe..
		// oldUmask = syscall.Umask(022)
		trainFile, err := os.OpenFile(trainPath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Printf("Error opening train file: %v\n", err)
		}
		defer trainFile.Close()
		// _ = syscall.Umask(oldUmask)

		// copy Train file contents to disk
		n_train, err := io.Copy(trainFile, trainTempFile)
		if err != nil {
			log.Printf("Error copying train file to disk: %v\n", err)
		}

		dataTempFile, dataHandler, err := r.FormFile("DataDir")
		if err != nil {
			log.Printf("Error reading data form file: %v\n", err)
		}
		defer dataTempFile.Close()

		// create new file to save down Data Dir on disk
		dataPath := userDir + filepath.Base(dataHandler.Filename)
		dataFile, err := os.OpenFile(dataPath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Printf("Error opening data file: %v\n", err)
		}
		defer dataFile.Close()
		defer os.Remove(dataPath)

		// copy Data Dir contents to disk
		n_data, err := io.Copy(dataFile, dataTempFile)
		if err != nil {
			log.Printf("Error copying data file to disk: %v\n", err)
		}

		// untar/gzip Data Dir
		err = archiver.TarGz.Open(dataPath, userDir)
		if err != nil {
			log.Printf("Error unzipping data dir: %v\n", err)
		}

		// send response to client
		n := n_train + n_data + n_requirements
		io.WriteString(w, fmt.Sprintf("%d bytes recieved and saved.\n", n))

		// execute train.py
		venv := "venv-" + username
		// TODO: make safer..?
		// log.Printf("Executing: python %s\n", trainPath)
		// trainCmd := exec.Command("python", trainPath)
		// TODO: files downloaded from web in trainPath are ending up in server folder; should be executing from more isolated environment in user folder
		longCmdString := fmt.Sprintf("source /usr/local/bin/virtualenvwrapper.sh; mkvirtualenv -r %s %s; python %s; deactivate; rmvirtualenv %s",
			requirementsPath, venv, trainPath, venv)
		log.Printf("Executing: %s\n", longCmdString)
		trainCmd := exec.Command("bash", "-c", longCmdString)
		trainOut, err := trainCmd.Output()
		if err != nil {
			log.Printf("Error executing %s: %v\n", longCmdString, err)
			io.WriteString(w, fmt.Sprintf("Failure executing %s\n", trainHandler.Filename))
		} else {
			log.Printf("Output: \n%s\n", string(trainOut))
			io.WriteString(w, string(trainOut))
		}
	} else {
		log.Printf("Upload received non-POST method.\n")
		io.WriteString(w, "Upload only receives POSTs.\n")
	}
}
