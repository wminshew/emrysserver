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

	// inbound bids from miners
	Bids map[uuid.UUID]chan *job.Bid
}

// InitPool creates a new Pool of miner connections
func InitPool() {
	Pool = &pool{
		miners:     make(map[*miner]bool),
		miner:      make(map[uuid.UUID]*miner),
		register:   make(chan *miner),
		unregister: make(chan *miner),
		messages:   make(chan []byte),
		Bids:       make(map[uuid.UUID]chan *job.Bid),
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
	// 	var winBid *job.Bid
	// 	j.PayRate = math.Inf(1)
	// auction:
	// 	for {
	// 		select {
	// 		// TODO: how to handle single miner bidding multiple times on single job? on multiple jobs?
	// 		case b := <-p.Bids[j.ID]:
	// 			n++
	// 			if winBid == nil {
	// 				winBid = b
	// 			} else if b.MinRate < winBid.MinRate {
	// 				j.PayRate = winBid.MinRate
	// 				winBid = b
	// 			} else if b.MinRate < j.PayRate {
	// 				j.PayRate = b.MinRate
	// 			}
	// 			go func() {
	// 				if _, err = db.Db.Query("INSERT INTO bids (bid_uuid, job_uuid, miner_uuid, min_rate, late) VALUES ($1, $2, $3, $4, $5)",
	// 					b.ID, b.JobID, b.MinerID, b.MinRate, false); err != nil {
	// 					log.Printf("Error inserting bid into db: %v\n", err)
	// 					return
	// 				}
	// 			}()
	// 		case <-time.After(5 * time.Second):
	// 			p.Bids[j.ID] = nil
	// 			log.Printf("Bidding complete!\n")
	// 			break auction
	// 		}
	// 	}
	//
	// 	if winBid == nil {
	// 		log.Printf("No bids received!\n")
	// 		return
	// 	}
	// 	if math.IsInf(j.PayRate, 1) {
	// 		j.PayRate = winBid.MinRate
	// 	}
	// 	log.Printf("%d bid(s) received.\n", n)
	// 	log.Printf("Highest bid: %+v\n", winBid.ID)
	// 	log.Printf("Pay rate: %v\n", j.PayRate)
	// 	log.Printf("Notifying winner %v\n", winBid.MinerID)
	// finMsg <- []byte("Miner auction success! Winning bidder selected.\n")
	// finMsg = nil
}
