package miner

import (
	"bytes"
	"encoding/json"
	"github.com/satori/go.uuid"
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

	// maps UUIDs to miners
	miner map[uuid.UUID]*miner

	// register requests from miners
	register chan *miner

	// unregister requests from miners
	unregister chan *miner

	// outbound messages to miners
	messages chan []byte

	// job auctions
	auctions map[uuid.UUID]*auction
}

// InitPool creates a new Pool of miner connections
func InitPool() {
	Pool = &pool{
		miners:     make(map[*miner]bool),
		miner:      make(map[uuid.UUID]*miner),
		register:   make(chan *miner),
		unregister: make(chan *miner),
		messages:   make(chan []byte),
		auctions:   make(map[uuid.UUID]*auction),
	}
}

// RunPool manages the Pool
func RunPool() {
	for {
		select {
		case miner := <-Pool.register:
			Pool.miners[miner] = true
			Pool.miner[miner.ID] = miner
		case miner := <-Pool.unregister:
			if _, ok := Pool.miners[miner]; ok {
				delete(Pool.miners, miner)
				delete(Pool.miner, miner.ID)
				close(miner.sendMsg)
			}
		case m := <-Pool.messages:
			for miner := range Pool.miners {
				select {
				case miner.sendMsg <- m:
				default:
					delete(Pool.miners, miner)
					delete(Pool.miner, miner.ID)
					close(miner.sendMsg)
				}
			}
		}
	}
}

func (p *pool) AuctionJob(j *job.Job) {
	a := newAuction(j.ID)
	go a.run(p)

	m := job.Message{
		Message: "New job posted!",
		Job:     j,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	err := enc.Encode(m)
	if err != nil {
		log.Printf("Error encoding new job message: %v\n", err)
		return
	}
	p.messages <- buf.Bytes()

	// winBid := a.winner()
}
