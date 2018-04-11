package handlers

import (
	"context"
	"docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"docker.io/go-docker/api/types/container"
	"encoding/json"
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
			log.Printf("Error parsing request: %v\n", err)
			return
		}

		// TODO: add uuid or some other unique identifier for users [emails can't be used in paths safely]
		username := "test2"

		// TODO: re-factor job processing; take out file saving, add relevant paths to r.context
		// TODO: add extra directory layer for project/job number (git vcs?); return job number to client
		userDir := filepath.Join("user-upload", username)
		if err = os.MkdirAll(userDir, 0755); err != nil {
			log.Printf("Error creating user directory %s: %v\n", userDir, err)
			return
		}

		requirementsTempFile, requirementsHandler, err := r.FormFile("requirements")
		if err != nil {
			log.Printf("Error reading requirements form file: %v\n", err)
			return
		}
		defer requirementsTempFile.Close()
		requirementsPath := filepath.Join(userDir, filepath.Base(requirementsHandler.Filename))
		requirementsFile, err := os.OpenFile(requirementsPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("Error opening requirements file: %v\n", err)
			return
		}
		defer requirementsFile.Close()
		n_requirements, err := io.Copy(requirementsFile, requirementsTempFile)
		if err != nil {
			log.Printf("Error copying requirements file to disk: %v\n", err)
			return
		}

		trainTempFile, trainHandler, err := r.FormFile("train")
		if err != nil {
			log.Printf("Error reading train form file: %v\n", err)
			return
		}
		defer trainTempFile.Close()
		trainPath := filepath.Join(userDir, filepath.Base(trainHandler.Filename))
		trainFile, err := os.OpenFile(trainPath, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			log.Printf("Error opening train file: %v\n", err)
			return
		}
		defer trainFile.Close()
		n_train, err := io.Copy(trainFile, trainTempFile)
		if err != nil {
			log.Printf("Error copying train file to disk: %v\n", err)
			return
		}

		dataTempFile, dataHandler, err := r.FormFile("data")
		if err != nil {
			log.Printf("Error reading data form file: %v\n", err)
			return
		}
		defer dataTempFile.Close()
		dataPath := filepath.Join(userDir, filepath.Base(dataHandler.Filename))
		dataFile, err := os.OpenFile(dataPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("Error opening data file: %v\n", err)
			return
		}
		defer dataFile.Close()
		defer os.Remove(dataPath)
		n_data, err := io.Copy(dataFile, dataTempFile)
		if err != nil {
			log.Printf("Error copying data file to disk: %v\n", err)
			return
		}
		err = archiver.TarGz.Open(dataPath, userDir)
		if err != nil {
			log.Printf("Error unzipping data dir: %v\n", err)
			return
		}

		n := n_train + n_data + n_requirements
		io.WriteString(w, fmt.Sprintf("%d bytes recieved and saved.\n", n))

		ctx := context.Background()
		cli, err := docker.NewEnvClient()
		if err != nil {
			log.Print(err)
			return
		}

		linkedDocker := filepath.Join(userDir, "Dockerfile")
		err = os.Link("Dockerfile.user", linkedDocker)
		if err != nil {
			log.Print(err)
			return
		}
		defer os.Remove(linkedDocker)
		buildCtxPath := filepath.Join(userDir + ".tar.gz")
		ctxFiles, err := filepath.Glob(filepath.Join(userDir, "/*"))
		if err != nil {
			log.Print(err)
			return
		}
		err = archiver.TarGz.Make(buildCtxPath, ctxFiles)
		if err != nil {
			log.Print(err)
			return
		}
		defer os.Remove(buildCtxPath)
		buildCtx, err := os.Open(buildCtxPath)
		if err != nil {
			log.Print(err)
			return
		}
		defer buildCtx.Close()
		buildResp, err := cli.ImageBuild(ctx, buildCtx, types.ImageBuildOptions{
			// TODO: add tags for emrys / project / job?
			Tags: []string{username},
			// TODO: explore Isolation: types.Isolation.IsHyperV
			BuildArgs: map[string]*string{
				"USER": &username,
			},
		})
		if err != nil {
			log.Print(err)
			return
		}
		defer buildResp.Body.Close()

		printBuildStream(buildResp.Body)

		resp, err := cli.ContainerCreate(ctx, &container.Config{
			Image: username,
		}, nil, nil, "")
		if err != nil {
			log.Print(err)
			return
		}

		if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
			log.Print(err)
			return
		}
		defer cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})

		statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				log.Print(err)
				return
			}
		case <-statusCh:
		}

		out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
		if err != nil {
			log.Print(err)
			return
		}

		tee := io.TeeReader(out, w)
		_, err = io.Copy(os.Stdout, tee)
		if err != nil && err != io.EOF {
			log.Print(err)
			return
		}

	} else {
		log.Printf("Upload received non-POST method.\n")
		io.WriteString(w, "Upload only receives POSTs.\n")
	}
}

func printBuildStream(r io.Reader) {
	type Stream struct {
		Stream string
	}
	dec := json.NewDecoder(r)
	for dec.More() {
		var s Stream
		err := dec.Decode(&s)
		if err != nil {
			log.Print(err)
		}

		fmt.Printf("%v", s.Stream)
	}
}
