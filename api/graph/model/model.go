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
	Upvotes   int    `json:"upvotes"`
}

type CalorieQuery struct {
	ID              string  `json:"id"`
	MealText        string  `json:"mealText"`
	Model           string  `json:"model"`
	ReasoningEffort *string `json:"reasoningEffort"`
	Calories        *int    `json:"calories"`
	ResponseTimeMs  *int    `json:"responseTimeMs"`
	TotalTokens     *int    `json:"totalTokens"`
	Status          string  `json:"status"`
	ErrorMessage    *string `json:"errorMessage"`
	CreatedAt       string  `json:"createdAt"`
}

type JobResult struct {
	MessageID string `json:"messageId"`
	Room      string `json:"room"`
	Sender    string `json:"sender"`
	Result    int    `json:"result"`
	Timestamp string `json:"timestamp"`
}
