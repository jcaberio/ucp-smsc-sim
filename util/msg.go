package util

// Message is latest message to be displayed in the web UI.
type Message struct {
	Message   string `json:"message"`
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Timestamp string `json:"timestamp"`
}
