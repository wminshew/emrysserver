package job

import (
	"github.com/satori/go.uuid"
	"io"
	"sync"
)

type pipe struct {
	pr *io.PipeReader
	pw *io.PipeWriter
}

var (
	logPipes = make(map[uuid.UUID]*pipe)
	muLog    = &sync.Mutex{}
	dirPipes = make(map[uuid.UUID]*pipe)
	muDir    = &sync.Mutex{}
)

func getLogPipe(u uuid.UUID) *pipe {
	return getPipe(muLog, logPipes, u)
}

func deleteLogPipe(u uuid.UUID) {
	delete(logPipes, u)
}

func getDirPipe(u uuid.UUID) *pipe {
	return getPipe(muDir, dirPipes, u)
}

func deleteDirPipe(u uuid.UUID) {
	delete(dirPipes, u)
}

func getPipe(mu *sync.Mutex, pipes map[uuid.UUID]*pipe, u uuid.UUID) *pipe {
	mu.Lock()
	defer mu.Unlock()
	if pipes[u] == nil {
		pr, pw := io.Pipe()
		pipes[u] = &pipe{
			pr: pr,
			pw: pw,
		}
	}
	return pipes[u]
}
