package slack

const (
	maxRetries = 10
)

type slackMessage struct {
	Text string `json:"text,omitempty"`
}
