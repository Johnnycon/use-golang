package model

type Room struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Message struct {
	ID        string `json:"id"`
	Room      string `json:"room"`
	Sender    string `json:"sender"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
}

type JobResult struct {
	MessageID string `json:"messageId"`
	Room      string `json:"room"`
	Sender    string `json:"sender"`
	Result    int    `json:"result"`
	Timestamp string `json:"timestamp"`
}
