package user

type contextKey string

func (c contextKey) String() string {
	return "user context key " + string(c)
}
