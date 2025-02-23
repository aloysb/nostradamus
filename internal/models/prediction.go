package models

// Prediction represents a single event prediction
type Prediction struct {
	Timeframe   string `json:"timeframe"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}

// PredictionResponse represents the initial predictions response
type PredictionResponse struct {
	OriginalPrompt string       `json:"original_prompt"`
	Predictions    []Prediction `json:"predictions"`
}
