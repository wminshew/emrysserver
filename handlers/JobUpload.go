package handlers

import (
	"context"
	"docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"docker.io/go-docker/api/types/container"
	"encoding/json"
	"fmt"
	"github.com/mholt/archiver"
	"github.com/wminshew/check"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// JobUpload handles python job posted by user
func JobUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		fw := newFlushWriter(w)
		_, err := fw.Write([]byte("Unpacking request...\n"))
		if err != nil {
			log.Printf("Error writing to flushWriter: %v\n", err)
		}
		maxMemory := int64(1) << 31
		err = r.ParseMultipartForm(maxMemory)
		if err != nil {
			log.Printf("Error parsing request: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// TODO: add uuid or some other unique identifier for users [emails can't be used in paths safely]
		username := "test2"

		// TODO: re-factor job processing; take out file saving, add relevant paths to r.context
		// TODO: add extra directory layer for project/job number (git vcs?); return job number to client
		userDir := filepath.Join("user-upload", username)
		if err = os.MkdirAll(userDir, 0755); err != nil {
			log.Printf("Error creating user directory %s: %v\n", userDir, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		requirementsTempFile, requirementsHeader, err := r.FormFile("requirements")
		if err != nil {
			log.Printf("Error reading requirements form file: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer check.Err(requirementsTempFile.Close)
		requirementsPath := filepath.Join(userDir, filepath.Base(requirementsHeader.Filename))
		requirementsFile, err := os.OpenFile(requirementsPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Error opening requirements file: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer check.Err(requirementsFile.Close)
		_, err = io.Copy(requirementsFile, requirementsTempFile)
		if err != nil {
			log.Printf("Error copying requirements file to disk: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		trainTempFile, trainHeader, err := r.FormFile("train")
		if err != nil {
			log.Printf("Error reading train form file: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer check.Err(trainTempFile.Close)
		trainPath := filepath.Join(userDir, filepath.Base(trainHeader.Filename))
		trainFile, err := os.OpenFile(trainPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			log.Printf("Error opening train file: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer check.Err(trainFile.Close)
		_, err = io.Copy(trainFile, trainTempFile)
		if err != nil {
			log.Printf("Error copying train file to disk: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		dataTempFile, dataHeader, err := r.FormFile("data")
		if err != nil {
			log.Printf("Error reading data form file: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer check.Err(dataTempFile.Close)
		dataPath := filepath.Join(userDir, filepath.Base(dataHeader.Filename))
		dataFile, err := os.OpenFile(dataPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Error opening data file: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer check.Err(dataFile.Close)
		defer check.Err(func() error { return os.Remove(dataPath) })
		_, err = io.Copy(dataFile, dataTempFile)
		if err != nil {
			log.Printf("Error copying data file to disk: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// TODO: need to remove old data dir contents / properly manage data update from
		// last job using git lfs or rsync or something
		err = archiver.TarGz.Open(dataPath, userDir)
		if err != nil {
			log.Printf("Error unzipping data dir: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = fw.Write([]byte("Building image...\n"))
		if err != nil {
			log.Printf("Error writing to flushWriter: %v\n", err)
		}

		ctx := context.Background()
		cli, err := docker.NewEnvClient()
		if err != nil {
			log.Printf("Error creating new docker client: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		linkedDocker := filepath.Join(userDir, "Dockerfile")
		userDockerfile := filepath.Join("Dockerfiles", "Dockerfile.user")
		err = os.Link(userDockerfile, linkedDocker)
		if err != nil {
			log.Printf("Error linking dockerfile into user directory: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer check.Err(func() error { return os.Remove(linkedDocker) })
		buildCtxPath := filepath.Join(userDir + ".tar.gz")
		ctxFiles, err := filepath.Glob(filepath.Join(userDir, "/*"))
		if err != nil {
			log.Printf("Error collecting docker context files: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = archiver.TarGz.Make(buildCtxPath, ctxFiles)
		if err != nil {
			log.Printf("Error archiving docker context files: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer check.Err(func() error { return os.Remove(buildCtxPath) })
		buildCtx, err := os.Open(buildCtxPath)
		if err != nil {
			log.Printf("Error opening archived docker context files: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		userHome := "/home/user"
		buildResp, err := cli.ImageBuild(ctx, buildCtx, types.ImageBuildOptions{
			BuildArgs: map[string]*string{
				"HOME": &userHome,
			},
			ForceRemove: true,
			// TODO: add more tags or labels for emrys / project / job?
			Tags: []string{username},
			// Labels: map[string]string{}
		})
		if err != nil {
			log.Printf("Error building image: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer check.Err(buildResp.Body.Close)
		// removing the image immediately after run means no caching
		// defer check.Err(func() error {
		// 	_, err := cli.ImageRemove(ctx, username, types.ImageRemoveOptions{
		// 		Force: true,
		// 	})
		// 	return err
		// })

		printBuildStream(buildResp.Body)
		_, err = fw.Write([]byte("Running image...\n"))
		if err != nil {
			log.Printf("Error writing to flushWriter: %v\n", err)
		}

		// TODO: do I need to preserve users' file structure?
		// [relative pathing between train.py and path/to/data/]
		wd, err := os.Getwd()
		if err != nil {
			log.Printf("Error getting working directory: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hostDataPath := filepath.Join(wd, userDir, "data")
		dockerDataPath := filepath.Join(userHome, "data")
		resp, err := cli.ContainerCreate(ctx, &container.Config{
			Image: username,
			Tty:   true,
		}, &container.HostConfig{
			AutoRemove: true,
			Binds: []string{
				fmt.Sprintf("%s:%s", hostDataPath, dockerDataPath),
			},
			Runtime: "nvidia",
		}, nil, "")
		if err != nil {
			log.Printf("Error creating container: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// TODO: how do I balance long jobs & container timeout?
		if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
			log.Printf("Error starting container: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{
			Follow:     true,
			ShowStdout: true,
		})
		if err != nil {
			log.Printf("Error logging container: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tee := io.TeeReader(out, fw)
		_, err = io.Copy(os.Stdout, tee)
		if err != nil && err != io.EOF {
			log.Printf("Error copying to stdout: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
		// select {
		// case err := <-errCh:
		// 	if err != nil {
		// 		log.Print(err)
		// 		return
		// 	}
		// case <-statusCh:
		// }

	} else {
		log.Printf("Upload received non-POST method.\n")
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("Upload only receives POST.\n"))
		if err != nil {
			log.Printf("Error writing to http.ResponseWriter: %v\n", err)
		}
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
			log.Printf("Error decoding json build stream: %v\n", err)
		}

		fmt.Printf("%v", s.Stream)
	}
}
