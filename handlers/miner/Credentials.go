package miner

type Credentials struct {
	Email    string `json:"email", db:"email"`
	Password string `json:"password", db:"password"`
}
