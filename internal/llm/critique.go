package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"nostradamus/internal/config"
	"nostradamus/internal/logger"
)

// GenerateCritiquedPredictions calls the LLM API to first generate predictions and then critiques them.
// It retries up to 10 times until it gets a valid JSON response.
func GenerateCritiquedPredictions(input string, client *http.Client) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", errors.New("no input provided")
	}

	// Build the initial prediction prompt.
	predictionPrompt := fmt.Sprintf("You are a predictor of future stock market events. Given the event: %q, generate predictions in JSON format. The JSON output must have \"original_prompt\" equal to the input and \"predictions\" be an array with between 1 and 10 items with each item containing \"timeframe\", \"description\", and \"impact\".", input)
	initialResponse, err := client.CallLLM(predictionPrompt)
	if err != nil {
		return "", err
	}

	var lastErr error
	for attempt := 1; attempt <= 10; attempt++ {
		// Build the critique prompt using the initial predictions.
		critiquePrompt := fmt.Sprintf("You are a knowledgeable investor. Critically review the following predictions in JSON format and add two additional fields to each prediction: \"confidence\" (a float between 0 and 1) and \"critique\" (a string explaining why this prediction is likely or not). Input predictions: %s", initialResponse)
		critiqueResponse, err := client.CallLLM(critiquePrompt)
		if err != nil {
			logger.Logger.Error("Critique API call failed", "attempt", attempt, "error", err)
			lastErr = err
			time.Sleep(config.RetryDelay)
			continue
		}

		// Validate JSON response
		var tmp map[string]interface{}
		if err = json.Unmarshal([]byte(critiqueResponse), &tmp); err != nil {
			logger.Logger.Error("Invalid JSON in critique response", "attempt", attempt, "error", err)
			lastErr = err
			time.Sleep(config.RetryDelay)
			continue
		}

		return critiqueResponse, nil
	}
	return "", fmt.Errorf("failed after 10 critique attempts: last error: %v", lastErr)
}
