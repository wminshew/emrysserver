package miner

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
				close(miner.send)
			}
		case job := <-Pool.jobs:
			for miner := range Pool.miners {
				select {
				case miner.send <- job:
				default:
					close(miner.send)
					delete(Pool.miners, miner)
				}
			}
		}
	}
}

func (p *pool) NewJob(newJob []byte) {
	p.jobs <- newJob
}
