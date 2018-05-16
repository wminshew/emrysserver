package miner

import (
	"bytes"
	"compress/zlib"
	"encoding/gob"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/db"
	"io"
	"log"
	"math"
	"time"
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

	// inbound jobs from users
	jobs chan []byte

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
		jobs:       make(chan []byte),
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
				close(miner.sendJob)
				close(miner.sendText)
				close(miner.sendImg)
			}
		case j := <-Pool.jobs:
			for miner := range Pool.miners {
				select {
				case miner.sendJob <- j:
				default:
					delete(Pool.miners, miner)
					delete(Pool.miner, miner.ID)
					close(miner.sendJob)
					close(miner.sendText)
					close(miner.sendImg)
				}
			}
		}
	}
}

func (p *pool) AuctionJob(j *job.Job, finMsg chan []byte, sendImg chan *io.ReadCloser) {
	defer func() {
		if finMsg != nil {
			finMsg <- []byte("Miner auction failed. Please try again.\n")
		}
	}()
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
	p.Bids[j.ID] = make(chan *job.Bid)
	n := 0
	p.jobs <- buf.Bytes()
	// TODO: add cloud providers before to avoid nil issues? or are they just regular miners. Probably the latter..
	var winBid *job.Bid
	j.PayRate = math.Inf(1)

auction:
	for {
		select {
		// TODO: make sure no double bidding?
		case b := <-p.Bids[j.ID]:
			n++
			if winBid == nil {
				winBid = b
			} else if b.MinRate < winBid.MinRate {
				j.PayRate = winBid.MinRate
				winBid = b
			} else if b.MinRate < j.PayRate {
				j.PayRate = b.MinRate
			}
			go func() {
				if _, err = db.Db.Query("INSERT INTO bids (bid_uuid, job_uuid, miner_uuid, min_rate) VALUES ($1, $2, $3, $4)",
					b.ID, b.JobID, b.MinerID, b.MinRate); err != nil {
					log.Printf("Error inserting bid into db: %v\n", err)
					return
				}
			}()
		case <-time.After(5 * time.Second):
			p.Bids[j.ID] = nil
			log.Printf("Bidding complete!\n")
			break auction
		}
	}

	if winBid == nil {
		log.Printf("No bids received!\n")
		return
	}
	if math.IsInf(j.PayRate, 1) {
		j.PayRate = winBid.MinRate
	}
	log.Printf("%d bid(s) received.\n", n)
	log.Printf("Highest bid: %+v\n", winBid.ID)
	log.Printf("Pay rate: %v\n", j.PayRate)
	log.Printf("Notifying winner %v\n", winBid.MinerID)
	p.miner[winBid.MinerID].sendText <- []byte("You won!\n")
	finMsg <- []byte("Miner auction success! Winning bidder selected.\n")
	finMsg = nil
	log.Printf("Sending image to winner\n")
	p.miner[winBid.MinerID].sendText <- []byte("Image\n")
	p.miner[winBid.MinerID].sendImg <- <-sendImg
	log.Printf("Sending data to winner\n")
	p.miner[winBid.MinerID].sendText <- []byte("Data\n")
	// p.miner[winBid.MinerID].sendData <- &data
	// TODO: insert job into DB; maybe do in JobUpload
	// go func() {
	// 	if _, err = db.Db.Query("INSERT INTO bids (bids_uuid, job_uuid, miner_uuid, min_rate) VALUES ($1, $2, $3, $4)",
	// 	b.ID, b.JobID, b.MinerID, b.MinRate); err != nil {
	// 		log.Printf("Error inserting bid into db: %v\n", err)
	// 		return
	// 	}
	// }()
	// TODO: pass something back so image can be sent ???
}
