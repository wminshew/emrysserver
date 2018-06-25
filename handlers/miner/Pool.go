package miner

import (
	"bytes"
	"encoding/json"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/pkg/app"
)

var p *pool

type pool struct {
	miners     map[*miner]bool
	miner      map[uuid.UUID]*miner
	register   chan *miner
	unregister chan *miner
	messages   chan []byte
}

// InitPool creates a new pool of miner connections
func InitPool() {
	app.Sugar.Infof("Initializing miner pool...")
	p = &pool{
		miners:     make(map[*miner]bool),
		miner:      make(map[uuid.UUID]*miner),
		register:   make(chan *miner),
		unregister: make(chan *miner),
		messages:   make(chan []byte),
	}
}

// RunPool manages the pool
func RunPool() {
	for {
		select {
		case miner := <-p.register:
			p.miners[miner] = true
			p.miner[miner.ID] = miner
		case miner := <-p.unregister:
			if _, ok := p.miners[miner]; ok {
				delete(p.miners, miner)
				delete(p.miner, miner.ID)
				close(miner.sendMsg)
			}
		case m := <-p.messages:
			for miner := range p.miners {
				select {
				case miner.sendMsg <- m:
				default:
					delete(p.miners, miner)
					delete(p.miner, miner.ID)
					close(miner.sendMsg)
				}
			}
		}
	}
}

func (p *pool) auctionJob(j *job.Job) error {
	m := job.Message{
		Message: "New job posted!",
		Job:     j,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(m); err != nil {
		return err
	}
	p.messages <- buf.Bytes()

	return nil
}
