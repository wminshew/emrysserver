package miner

import (
	"bytes"
	"compress/zlib"
	"encoding/gob"
	"github.com/wminshew/emrys/pkg/job"
	"log"
)

// Pool manages the connections to non-working miners
var Pool *pool

// Pool maintains the set of active and available miners and
// broadcasts jobs to the miners
type pool struct {
	// registered miners
	miners map[*miner]bool

	// inbound jobs from users
	jobs chan []byte

	// register requests from miners
	register chan *miner

	// unregister requests from miners
	unregister chan *miner
}

// InitPool creates a new Pool of miner connections
func InitPool() {
	Pool = &pool{
		miners:     make(map[*miner]bool),
		jobs:       make(chan []byte),
		register:   make(chan *miner),
		unregister: make(chan *miner),
	}
}

// RunPool manages the Pool
func RunPool() {
	for {
		select {
		case miner := <-Pool.register:
			Pool.miners[miner] = true
		case miner := <-Pool.unregister:
			if _, ok := Pool.miners[miner]; ok {
				delete(Pool.miners, miner)
				close(miner.sendJob)
			}
		case j := <-Pool.jobs:
			for miner := range Pool.miners {
				select {
				case miner.sendJob <- j:
				default:
					close(miner.sendJob)
					delete(Pool.miners, miner)
				}
			}
		}
	}
}

func (p *pool) BroadcastJob(j *job.Job) {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	enc := gob.NewEncoder(zw)
	err := enc.Encode(j)
	if err != nil {
		log.Printf("Error encoding and compressing job: %v\n", err)
		return
	}
	err = zw.Close()
	if err != nil {
		log.Printf("Error closing zlib job writer: %v\n", err)
		return
	}
	// b := make([]byte, len(buf.Bytes()))
	// _ = copy(b, buf.Bytes())
	// log.Printf("Buffer: %v\n", b)
	log.Printf("Buffer: %+v\n", buf)
	p.jobs <- buf.Bytes()
}
