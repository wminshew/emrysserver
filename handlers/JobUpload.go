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
)

func JobUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		maxMemory := int64(1) << 31
		err := r.ParseMultipartForm(maxMemory)
		if err != nil {
			log.Printf("Error parsing request: %v\n", err)
		}

		// TODO: add uuid or some other unique identifier for users [emails can't be used in paths safely]
		username := "test2"
		// TODO: add extra directory layer for project/job number (git vcs?); return job number to client
		userDir := "./user-upload/" + username + "/"
		if err = os.MkdirAll(userDir, 0755); err != nil {
			log.Printf("Error creating user directory %s: %v\n", userDir, err)
		}

		requirementsTempFile, requirementsHandler, err := r.FormFile("requirements")
		if err != nil {
			log.Printf("Error reading requirements form file: %v\n", err)
		}
		defer requirementsTempFile.Close()
		requirementsPath := userDir + filepath.Base(requirementsHandler.Filename)
		requirementsFile, err := os.OpenFile(requirementsPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("Error opening requirements file: %v\n", err)
		}
		defer requirementsFile.Close()
		n_requirements, err := io.Copy(requirementsFile, requirementsTempFile)
		if err != nil {
			log.Printf("Error copying requirements file to disk: %v\n", err)
		}

		trainTempFile, trainHandler, err := r.FormFile("train")
		if err != nil {
			log.Printf("Error reading train form file: %v\n", err)
		}
		defer trainTempFile.Close()
		trainPath := userDir + filepath.Base(trainHandler.Filename)
		trainFile, err := os.OpenFile(trainPath, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			log.Printf("Error opening train file: %v\n", err)
		}
		defer trainFile.Close()
		n_train, err := io.Copy(trainFile, trainTempFile)
		if err != nil {
			log.Printf("Error copying train file to disk: %v\n", err)
		}

		dataTempFile, dataHandler, err := r.FormFile("data")
		if err != nil {
			log.Printf("Error reading data form file: %v\n", err)
		}
		defer dataTempFile.Close()
		dataPath := userDir + filepath.Base(dataHandler.Filename)
		dataFile, err := os.OpenFile(dataPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("Error opening data file: %v\n", err)
		}
		defer dataFile.Close()
		defer os.Remove(dataPath)
		n_data, err := io.Copy(dataFile, dataTempFile)
		if err != nil {
			log.Printf("Error copying data file to disk: %v\n", err)
		}
		err = archiver.TarGz.Open(dataPath, userDir)
		if err != nil {
			log.Printf("Error unzipping data dir: %v\n", err)
		}

		n := n_train + n_data + n_requirements
		io.WriteString(w, fmt.Sprintf("%d bytes recieved and saved.\n", n))

		venv := "venv-" + username
		// TODO: make safer..?
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
