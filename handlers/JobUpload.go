package handlers

import (
	"context"
	"docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"docker.io/go-docker/api/types/container"
	"fmt"
	"github.com/mholt/archiver"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func JobUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		maxMemory := int64(1) << 31
		err := r.ParseMultipartForm(maxMemory)
		if err != nil {
			log.Fatalf("Error parsing request: %v\n", err)
		}

		// TODO: add uuid or some other unique identifier for users [emails can't be used in paths safely]
		username := "test2"

		// TODO: re-factor job processing; take out file saving, add relevant paths to r.context
		// TODO: add extra directory layer for project/job number (git vcs?); return job number to client
		userDir := "./user-upload/" + username + "/"
		if err = os.MkdirAll(userDir, 0755); err != nil {
			log.Fatalf("Error creating user directory %s: %v\n", userDir, err)
		}

		requirementsTempFile, requirementsHandler, err := r.FormFile("requirements")
		if err != nil {
			log.Fatalf("Error reading requirements form file: %v\n", err)
		}
		defer requirementsTempFile.Close()
		requirementsPath := userDir + filepath.Base(requirementsHandler.Filename)
		requirementsFile, err := os.OpenFile(requirementsPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatalf("Error opening requirements file: %v\n", err)
		}
		defer requirementsFile.Close()
		n_requirements, err := io.Copy(requirementsFile, requirementsTempFile)
		if err != nil {
			log.Fatalf("Error copying requirements file to disk: %v\n", err)
		}

		trainTempFile, trainHandler, err := r.FormFile("train")
		if err != nil {
			log.Fatalf("Error reading train form file: %v\n", err)
		}
		defer trainTempFile.Close()
		trainPath := userDir + filepath.Base(trainHandler.Filename)
		trainFile, err := os.OpenFile(trainPath, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			log.Fatalf("Error opening train file: %v\n", err)
		}
		defer trainFile.Close()
		n_train, err := io.Copy(trainFile, trainTempFile)
		if err != nil {
			log.Fatalf("Error copying train file to disk: %v\n", err)
		}

		dataTempFile, dataHandler, err := r.FormFile("data")
		if err != nil {
			log.Fatalf("Error reading data form file: %v\n", err)
		}
		defer dataTempFile.Close()
		dataPath := userDir + filepath.Base(dataHandler.Filename)
		dataFile, err := os.OpenFile(dataPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatalf("Error opening data file: %v\n", err)
		}
		defer dataFile.Close()
		defer os.Remove(dataPath)
		n_data, err := io.Copy(dataFile, dataTempFile)
		if err != nil {
			log.Fatalf("Error copying data file to disk: %v\n", err)
		}
		err = archiver.TarGz.Open(dataPath, userDir)
		if err != nil {
			log.Fatalf("Error unzipping data dir: %v\n", err)
		}

		n := n_train + n_data + n_requirements
		io.WriteString(w, fmt.Sprintf("%d bytes recieved and saved.\n", n))

		venv := "venv-" + username
		longCmdString := fmt.Sprintf(`source /usr/local/bin/virtualenvwrapper.sh; \\
		mkvirtualenv -r %s %s; \\
		python %s; \\
		deactivate; \\
		rmvirtualenv %s`,
			requirementsPath, venv, trainPath, venv)
		// log.Printf("Executing: \n%s\n", longCmdString)
		// trainCmd := exec.Command("bash", "-c", longCmdString)
		// trainOut, err := trainCmd.Output()
		// if err != nil {
		// 	log.Fatalf("Error executing %s: %v\n", longCmdString, err)
		// 	io.WriteString(w, fmt.Sprintf("Failure executing %s\n", trainHandler.Filename))
		// } else {
		// 	log.Printf("Output: \n%s\n", string(trainOut))
		// 	io.WriteString(w, string(trainOut))
		// }

		log.Printf("Launching docker...\n")

		ctx := context.Background()
		cli, err := docker.NewEnvClient()
		if err != nil {
			log.Fatal(err)
		}

		// reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
		reader, err := cli.ImagePull(ctx, "ubuntu:16.04", types.ImagePullOptions{})
		// reader, err := cli.ImagePull(ctx, "floydhub/dl-docker", types.ImagePullOptions{})
		if err != nil {
			log.Fatal(err)
		}
		io.Copy(os.Stdout, reader)

		resp, err := cli.ContainerCreate(ctx, &container.Config{
			// Image: "dl-docker",
			// Image: "alpine",
			Image: "ubuntu:16.04",
			// Cmd:   []string{"echo", "hello world"},
			Cmd: []string{"/bin/bash", "-c", longCmdString},
			Tty: true,
		}, nil, nil, "")
		if err != nil {
			log.Fatal(err)
		}

		if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
			log.Fatal(err)
		}

		// statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
		statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				log.Fatal(err)
			}
		case <-statusCh:
		}

		out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.Copy(os.Stdout, out)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}

	} else {
		log.Printf("Upload received non-POST method.\n")
		io.WriteString(w, "Upload only receives POSTs.\n")
	}
}
