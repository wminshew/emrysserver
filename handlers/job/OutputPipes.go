package job

import (
	"github.com/satori/go.uuid"
	"io"
)

type pipe struct {
	r *io.PipeReader
	w *io.PipeWriter
}

var (
	logPipes = make(map[uuid.UUID]*pipe)
	dirPipes = make(map[uuid.UUID]*pipe)
)
