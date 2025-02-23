package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var retryDelay = 1 * time.Second

type Prediction struct {
	Timeframe   string `json:"timeframe"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}

type PredictionResponse struct {
	OriginalPrompt string       `json:"original_prompt"`
	Predictions    []Prediction `json:"predictions"`
}

type CritiquedPrediction struct {
	Timeframe   string  `json:"timeframe"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Confidence  float64 `json:"confidence"`
	Critique    string  `json:"critique"`
}

type CritiquedResponse struct {
	OriginalPrompt string               `json:"original_prompt"`
	Predictions    []CritiquedPrediction `json:"predictions"`
}

func main() {
	log.SetOutput(os.Stdout)
	if len(os.Args) < 2 {
		log.Println("Error: No input provided. Usage: go run main.go <input>")
		os.Exit(1)
	}
	input := strings.Join(os.Args[1:], " ")
	log.Printf("Received input: %s", input)
	result, err := generateCritiquedPredictions(input, http.DefaultClient)
	if err != nil {
		log.Printf("Error generating critiqued predictions: %v", err)
		os.Exit(1)
	}
	log.Printf("Final valid critiqued predictions:\n%s", result)
	fmt.Println(result)
}

func generatePredictions(input string, client *http.Client) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", errors.New("no input provided")
	}
	var lastError error
	// Try up to 10 attempts to get a valid response.
	for attempt := 1; attempt <= 10; attempt++ {
		log.Printf("Attempt %d: Calling LLM API for input: %s", attempt, input)
		responseStr, err := callLLM(client, input)
		if err != nil {
			log.Printf("Attempt %d: API call failed: %v", attempt, err)
			lastError = err
			time.Sleep(retryDelay)
			continue
		}
		log.Printf("Attempt %d: Received response: %s", attempt, responseStr)
		err = validateResponse([]byte(responseStr), input)
		if err != nil {
			log.Printf("Attempt %d: Validation failed: %v", attempt, err)
			lastError = err
			time.Sleep(retryDelay)
			continue
		}
		log.Printf("Attempt %d: Valid response received.", attempt)
		return responseStr, nil
	}
	return "", fmt.Errorf("failed after 10 attempts: last error: %v", lastError)
}

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
	log.Printf("Sending request to LLM API at %s with payload: %s", url, string(requestBody))

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

func callLLMCritique(client *http.Client, firstResult string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", errors.New("OPENAI_API_KEY is not set")
	}
	url := "https://api.openai.com/v1/chat/completions"

	critiquePrompt := fmt.Sprintf("You are a knowledgeable investor. Critically review the following predictions in JSON format and add two additional fields to each prediction: \"confidence\" (a float between 0 and 1) and \"critique\" (a string explaining why this prediction is likely or not). Ensure that the output JSON object has \"original_prompt\" equal to the original input and \"predictions\" is an array of prediction objects with the additional fields. Input predictions: %s", firstResult)

	requestPayload := map[string]interface{}{
		"model": "o1-mini",
		"messages": []map[string]string{
			{"role": "user", "content": critiquePrompt},
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
	log.Printf("Sending critique request to LLM API at %s with payload: %s", url, string(requestBody))

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
		return "", fmt.Errorf("critique API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

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

	// Fallback: try to unmarshal as a CritiquedResponse.
	var cr CritiquedResponse
	err = json.Unmarshal(bodyBytes, &cr)
	if err == nil && cr.OriginalPrompt != "" && len(cr.Predictions) >= 1 && len(cr.Predictions) <= 10 {
		resultBytes, err := json.Marshal(cr)
		if err != nil {
			return "", fmt.Errorf("failed to re-marshal CritiquedResponse: %v", err)
		}
		return string(resultBytes), nil
	}
	return "", fmt.Errorf("failed to parse LLM critique API response; fallback error: %v", err)
}

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

func generateCritiquedPredictions(input string, client *http.Client) (string, error) {
	// First, generate initial predictions.
	firstResult, err := generatePredictions(input, client)
	if err != nil {
		return "", err
	}

	var lastError error
	// Try up to 10 attempts for the critique agent.
	for attempt := 1; attempt <= 10; attempt++ {
		log.Printf("Critique Attempt %d: Calling critique LLM API with predictions.", attempt)
		critiqueResult, err := callLLMCritique(client, firstResult)
		if err != nil {
			log.Printf("Critique Attempt %d: API call failed: %v", attempt, err)
			lastError = err
			time.Sleep(retryDelay)
			continue
		}
		log.Printf("Critique Attempt %d: Received response: %s", attempt, critiqueResult)
		err = validateCritiqueResponse([]byte(critiqueResult), input)
		if err != nil {
			log.Printf("Critique Attempt %d: Validation failed: %v", attempt, err)
			lastError = err
			time.Sleep(retryDelay)
			continue
		}
		log.Printf("Critique Attempt %d: Valid critique response received.", attempt)
		return critiqueResult, nil
	}
	return "", fmt.Errorf("failed after 10 critique attempts: last error: %v", lastError)
}
