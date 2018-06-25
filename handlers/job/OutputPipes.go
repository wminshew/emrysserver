package job

import (
	"github.com/satori/go.uuid"
	"os"
	"sync"
)

type pipe struct {
	r *os.File
	w *os.File
}

var (
	logPipes = make(map[uuid.UUID]*pipe)
	muLog    = &sync.Mutex{}
	dirPipes = make(map[uuid.UUID]*pipe)
	muDir    = &sync.Mutex{}
)

func getLogPipe(u uuid.UUID) (*pipe, error) {
	return getPipe(muLog, logPipes, u)
}

func deleteLogPipe(u uuid.UUID) {
	delete(logPipes, u)
}

func getDirPipe(u uuid.UUID) (*pipe, error) {
	return getPipe(muDir, dirPipes, u)
}

func deleteDirPipe(u uuid.UUID) {
	delete(dirPipes, u)
}

func getPipe(mu *sync.Mutex, pipes map[uuid.UUID]*pipe, u uuid.UUID) (*pipe, error) {
	mu.Lock()
	defer mu.Unlock()
	if pipes[u] == nil {
		r, w, err := os.Pipe()
		if err != nil {
			return nil, err
		}
		pipes[u] = &pipe{
			r: r,
			w: w,
		}
	}
	return pipes[u], nil
}
