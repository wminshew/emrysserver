package miner

// Pool maintains the set of active and available miners and
// broadcasts jobs to the miners
type Pool struct {
	// registered miners
	miners map[*miner]bool

	// inbound jobs from users
	jobs chan []byte

	// register requests from miners
	register chan *miner

	// unregister requests from miners
	unregister chan *miner
}

// NewPool creates a new Pool of miner connections
func NewPool() *Pool {
	return &Pool{
		miners:     make(map[*miner]bool),
		jobs:       make(chan []byte),
		register:   make(chan *miner),
		unregister: make(chan *miner),
	}
}

// Run manages the Pool
func (p *Pool) Run() {
	for {
		select {
		case miner := <-p.register:
			p.miners[miner] = true
		case miner := <-p.unregister:
			if _, ok := p.miners[miner]; ok {
				delete(p.miners, miner)
				close(miner.send)
			}
		case job := <-p.jobs:
			for miner := range p.miners {
				select {
				case miner.send <- job:
				default:
					close(miner.send)
					delete(p.miners, miner)
				}
			}
		}
	}
}
