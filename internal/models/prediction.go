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

// CritiquedPrediction contains a prediction with critique info
type CritiquedPrediction struct {
	Timeframe   string  `json:"timeframe"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Confidence  float64 `json:"confidence"`
	Critique    string  `json:"critique"`
}

// CritiquedResponse represents the critiqued predictions response
type CritiquedResponse struct {
	OriginalPrompt string                `json:"original_prompt"`
	Predictions    []CritiquedPrediction `json:"predictions"`
}
