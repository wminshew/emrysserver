package handlers

// Credentials allow the server to authenticate a user or miner
type Credentials struct {
	Email    string `json:"email",db:"email"`
	Password string `json:"password",db:"password"`
}
