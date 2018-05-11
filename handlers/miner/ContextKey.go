package miner

type contextKey string

func (c contextKey) String() string {
	return "miner context key " + string(c)
}
