package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"nostradamus/internal/config"
	"nostradamus/internal/logger"
	"nostradamus/internal/llm"
	"nostradamus/internal/models"
)
type Prediction struct {
	Timeframe   string `json:"timeframe"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}

// PredictionResponse represents the initial predictions response structure from the LLM.
type PredictionResponse struct {
	OriginalPrompt string       `json:"original_prompt"`
	Predictions    []Prediction `json:"predictions"`
}

// CritiquedPrediction contains a prediction with additional critique information.
type CritiquedPrediction struct {
	Timeframe   string  `json:"timeframe"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Confidence  float64 `json:"confidence"`
	Critique    string  `json:"critique"`
}

// CritiquedResponse represents the critiqued predictions response structure.
type CritiquedResponse struct {
	OriginalPrompt string                `json:"original_prompt"`
	Predictions    []CritiquedPrediction `json:"predictions"`
}

// main is the entry point to the application. It parses input, calls the prediction generation,
// and prints the final critiqued predictions to standard output.
func main() {
	// Verify that input is provided.
	if len(os.Args) < 2 {
		logger.Error("No input provided", "usage", "go run main.go <input>")
		os.Exit(1)
	}
	input := strings.Join(os.Args[1:], " ")
	logger.Info("Received input", "input", input)

	// Generate critiqued predictions via LLM API calls.
	llmClient, err := llm.NewClient(http.DefaultClient)
	if err != nil {
		logger.Error("Error creating LLM client", "error", err)
		os.Exit(1)
	}
	result, err := llm.GenerateCritiquedPredictions(input, llmClient)
	if err != nil {
		logger.Error("Error generating critiqued predictions", "error", err)
		os.Exit(1)
	}
	logger.Info("Final valid critiqued predictions", "result", result)
	fmt.Println(result)
}

// generatePredictions calls the LLM API up to 10 times to generate predictions for the given input.
// It validates that the response adheres to the expected JSON structure and returns the valid response string.
func generatePredictions(input string, client *http.Client) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", errors.New("no input provided")
	}
	var lastError error
	// Attempt to obtain a valid prediction response up to 10 times.
	for attempt := 1; attempt <= 10; attempt++ {
		logger.Info("Attempting to call LLM API for predictions", "attempt", attempt, "input", input)
		responseStr, err := callLLM(client, input)
		if err != nil {
			logger.Error("LLM API call failed", "attempt", attempt, "error", err)
			lastError = err
			time.Sleep(retryDelay)
			continue
		}
		logger.Info("Received LLM API response", "attempt", attempt, "response", responseStr)
		err = validateResponse([]byte(responseStr), input)
		if err != nil {
			logger.Error("Validation failed for LLM API response", "attempt", attempt, "error", err)
			lastError = err
			time.Sleep(retryDelay)
			continue
		}
		logger.Info("Valid LLM API response received", "attempt", attempt)
		return responseStr, nil
	}
	return "", fmt.Errorf("failed after 10 attempts: last error: %v", lastError)
}

// callLLM sends a HTTP POST request to the LLM API with a prompt based on the given input and returns
// a sanitized response string. It first attempts to unmarshal the response as an LLM chat response,
// falling back to a direct PredictionResponse unmarshal if needed.
func callLLM(client *http.Client, input string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", errors.New("OPENAI_API_KEY is not set")
	}
	url := "https://api.openai.com/v1/chat/completions"

	finalPrompt := fmt.Sprintf("You are a predictor of future stock market events. Given the event: %q, generate predictions in JSON format. The JSON output must have the field \"original_prompt\" equal to the given input and \"predictions\" be an array with between 1 and 10 items, where each item contains \"timeframe\", \"description\", \"impact\". For example: { \"original_prompt\": %q, \"predictions\": [{ \"timeframe\": \"1 week\", \"description\": \"Prediction text\", \"impact\": \"market impact\" }] }", input, input)

	requestPayload := map[string]interface{}{
		"model": "o1-mini",
		"messages": []map[string]string{
			{"role": "user", "content": finalPrompt},
		},
	}
	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	logger.Info("Sending request to LLM API", "url", url, "payload", string(requestBody))

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Attempt to parse the response as an LLM chat response.
	type llmMessage struct {
		Content string `json:"content"`
	}
	type llmChoice struct {
		Message llmMessage `json:"message"`
	}
	type llmResponse struct {
		Choices []llmChoice `json:"choices"`
	}
	var lr llmResponse
	err = json.Unmarshal(bodyBytes, &lr)
	if err == nil && len(lr.Choices) > 0 {
		result := sanitizeResponse(lr.Choices[0].Message.Content)
		return result, nil
	}

	// Fallback: try to unmarshal as a PredictionResponse.
	var pr PredictionResponse
	err = json.Unmarshal(bodyBytes, &pr)
	if err == nil && pr.OriginalPrompt == input && len(pr.Predictions) >= 1 && len(pr.Predictions) <= 10 {
		resultBytes, err := json.Marshal(pr)
		if err != nil {
			return "", fmt.Errorf("failed to re-marshal PredictionResponse: %v", err)
		}
		return string(resultBytes), nil
	}
	return "", fmt.Errorf("failed to parse LLM API response; fallback error: %v", err)
}

// sanitizeResponse trims whitespace and removes markdown code fences from the response string, if present.
func sanitizeResponse(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) > 0 {
			lines = lines[1:]
		}
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
			lines = lines[:len(lines)-1]
		}
		s = strings.Join(lines, "\n")
		s = strings.TrimSpace(s)
	}
	return s
}

// validateResponse checks whether the given response JSON matches the expected PredictionResponse structure.
func validateResponse(response []byte, input string) error {
	var pr PredictionResponse
	err := json.Unmarshal(response, &pr)
	if err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}
	if pr.OriginalPrompt != input {
		return fmt.Errorf("original_prompt does not match input. Expected: %s, Got: %s", input, pr.OriginalPrompt)
	}
	if len(pr.Predictions) < 1 || len(pr.Predictions) > 10 {
		return fmt.Errorf("predictions length is out of range: got %d", len(pr.Predictions))
	}
	// Validate each prediction in the response.
	for i, p := range pr.Predictions {
		if strings.TrimSpace(p.Timeframe) == "" {
			return fmt.Errorf("prediction %d: timeframe is empty", i)
		}
		if strings.TrimSpace(p.Description) == "" {
			return fmt.Errorf("prediction %d: description is empty", i)
		}
		if strings.TrimSpace(p.Impact) == "" {
			return fmt.Errorf("prediction %d: impact is empty", i)
		}
	}
	return nil
}


// validateCritiqueResponse verifies that the critique response JSON conforms to the expected CritiquedResponse structure.
func validateCritiqueResponse(response []byte, input string) error {
	var cr CritiquedResponse
	err := json.Unmarshal(response, &cr)
	if err != nil {
		return fmt.Errorf("invalid JSON in critique response: %v", err)
	}
	if cr.OriginalPrompt != input {
		return fmt.Errorf("original_prompt does not match input. Expected: %s, Got: %s", input, cr.OriginalPrompt)
	}
	if len(cr.Predictions) < 1 || len(cr.Predictions) > 10 {
		return fmt.Errorf("predictions array length is out of range: %d", len(cr.Predictions))
	}
	// Validate each critiqued prediction.
	for i, p := range cr.Predictions {
		if strings.TrimSpace(p.Timeframe) == "" {
			return fmt.Errorf("prediction %d: timeframe is empty", i)
		}
		if strings.TrimSpace(p.Description) == "" {
			return fmt.Errorf("prediction %d: description is empty", i)
		}
		if strings.TrimSpace(p.Impact) == "" {
			return fmt.Errorf("prediction %d: impact is empty", i)
		}
		if p.Confidence < 0 || p.Confidence > 1 {
			return fmt.Errorf("prediction %d: confidence %f is out of range", i, p.Confidence)
		}
		if strings.TrimSpace(p.Critique) == "" {
			return fmt.Errorf("prediction %d: critique is empty", i)
		}
	}
	return nil
}

// generateCritiquedPredictions first obtains the initial predictions from the LLM API and then refines them
// by calling the critique API. It returns the final, validated critiqued predictions as a JSON string.
func generateCritiquedPredictions(input string, client *http.Client) (string, error) {
	// Generate initial predictions.
	initialPredictions, err := generatePredictions(input, client)
	if err != nil {
		return "", err
	}

	var lastError error
	// Attempt up to 10 times to obtain a valid critique response.
	for attempt := 1; attempt <= 10; attempt++ {
		logger.Info("Attempting to call critique LLM API", "attempt", attempt)
		critiqueResult, err := callLLMCritique(client, initialPredictions)
		if err != nil {
			logger.Error("Critique API call failed", "attempt", attempt, "error", err)
			lastError = err
			time.Sleep(retryDelay)
			continue
		}
		logger.Info("Received critique response", "attempt", attempt, "response", critiqueResult)
		err = validateCritiqueResponse([]byte(critiqueResult), input)
		if err != nil {
			logger.Error("Critique response validation failed", "attempt", attempt, "error", err)
			lastError = err
			time.Sleep(retryDelay)
			continue
		}
		logger.Info("Valid critique response received", "attempt", attempt)
		return critiqueResult, nil
	}
	return "", fmt.Errorf("failed after 10 critique attempts: last error: %v", lastError)
}
