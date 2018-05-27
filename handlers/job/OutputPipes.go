package job

import (
	"github.com/satori/go.uuid"
	"io"
)

type pipe struct {
	pr *io.PipeReader
	pw *io.PipeWriter
}

// outputLog facilitate output log transfer from miner to user via server
var outputLog map[uuid.UUID]*pipe = make(map[uuid.UUID]*pipe)

// outputDir facilitate output log transfer from miner to user via server
var outputDir map[uuid.UUID]*pipe = make(map[uuid.UUID]*pipe)
