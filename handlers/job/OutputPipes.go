package job

import (
	"github.com/satori/go.uuid"
	"io"
)

type pipe struct {
	pr *io.PipeReader
	pw *io.PipeWriter
}

// outputPipes facilitate output transfer from miner to user via server
var outputPipes map[uuid.UUID]*pipe = make(map[uuid.UUID]*pipe)
