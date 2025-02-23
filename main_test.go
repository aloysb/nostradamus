package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// Dummy variable to force inclusion of symbols from main.go.
// This ensures that generatePredictions, retryDelay, PredictionResponse, and Prediction
// are referenced, preventing "undefined" errors when linting this file alone.
var _ = func() interface{} {
	return struct {
		GeneratePredictions     func(string, *http.Client) (string, error)
		GenerateCritiquedPredictions func(string, *http.Client) (string, error)
		RetryDelay              time.Duration
		PredResponse            PredictionResponse
		Prediction              Prediction
	}{
		GeneratePredictions:     generatePredictions,
		GenerateCritiquedPredictions: generateCritiquedPredictions,
		RetryDelay:              retryDelay,
		PredResponse:            PredictionResponse{},
		Prediction:              Prediction{},
	}
}()

type RoundTripFunc func(req *http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGeneratePredictionsNoInput(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "testkey")
	client := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			t.Errorf("API should not be called when input is empty")
			return nil, nil
		}),
	}
	_, err := generatePredictions("", client)
	if err == nil {
		t.Error("Expected error for empty input, got nil")
	}
}

func TestAPIFailure(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "testkey")
	originalDelay := retryDelay
	retryDelay = 1 * time.Millisecond
	defer func() { retryDelay = originalDelay }()
	client := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("API down")
		}),
	}
	_, err := generatePredictions("test event", client)
	if err == nil || !strings.Contains(err.Error(), "API down") {
		t.Errorf("Expected API down error, got: %v", err)
	}
}

func TestValidResponse(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "testkey")
	originalDelay := retryDelay
	retryDelay = 1 * time.Millisecond
	defer func() { retryDelay = originalDelay }()

	// This is a PredictionResponse format (fallback branch) response.
	validResp := `{"original_prompt": "test event", "predictions": [{"timeframe": "1 week", "description": "Event A", "impact": "Market volatility"}]}`
	client := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(validResp)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	result, err := generatePredictions("test event", client)
	if err != nil {
		t.Errorf("Expected valid response, got error: %v", err)
	}

	var predResp PredictionResponse
	err = json.Unmarshal([]byte(result), &predResp)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	expectedResponse := PredictionResponse{
		OriginalPrompt: "test event",
		Predictions: []Prediction{
			{
				Timeframe:   "1 week",
				Description: "Event A",
				Impact:      "Market volatility",
			},
		},
	}

	if predResp.OriginalPrompt != expectedResponse.OriginalPrompt {
		t.Errorf("Expected original_prompt %s, got %s", expectedResponse.OriginalPrompt, predResp.OriginalPrompt)
	}
	if len(predResp.Predictions) != len(expectedResponse.Predictions) {
		t.Errorf("Expected %d predictions, got %d", len(expectedResponse.Predictions), len(predResp.Predictions))
	}
	for i, pred := range predResp.Predictions {
		if pred.Timeframe != expectedResponse.Predictions[i].Timeframe {
			t.Errorf("Prediction %d: Expected timeframe %s, got %s", i, expectedResponse.Predictions[i].Timeframe, pred.Timeframe)
		}
		if pred.Description != expectedResponse.Predictions[i].Description {
			t.Errorf("Prediction %d: Expected description %s, got %s", i, expectedResponse.Predictions[i].Description, pred.Description)
		}
		if pred.Impact != expectedResponse.Predictions[i].Impact {
			t.Errorf("Prediction %d: Expected impact %s, got %s", i, expectedResponse.Predictions[i].Impact, pred.Impact)
		}
	}

	expectedBytes, err := json.Marshal(expectedResponse)
	if err != nil {
		t.Fatalf("Failed to marshal expected response: %v", err)
	}
	expectedCompact := string(expectedBytes)
	if result != expectedCompact {
		t.Errorf("Expected response: %s, got: %s", expectedCompact, result)
	}
}

func TestInvalidJSONThenValid(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "testkey")
	originalDelay := retryDelay
	retryDelay = 1 * time.Millisecond
	defer func() { retryDelay = originalDelay }()

	callCount := 0
	validResp := `{"original_prompt": "test event", "predictions": [{"timeframe": "1 week", "description": "Event A", "impact": "Market volatility"}]}`
	client := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount < 3 {
				// Return invalid JSON on first two attempts
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("invalid json")),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(validResp)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	result, err := generatePredictions("test event", client)
	if err != nil {
		t.Errorf("Expected valid response after retries, got error: %v", err)
	}

	var predResp PredictionResponse
	err = json.Unmarshal([]byte(result), &predResp)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	expectedResponse := PredictionResponse{
		OriginalPrompt: "test event",
		Predictions: []Prediction{
			{
				Timeframe:   "1 week",
				Description: "Event A",
				Impact:      "Market volatility",
			},
		},
	}

	if predResp.OriginalPrompt != expectedResponse.OriginalPrompt {
		t.Errorf("Expected original_prompt %s, got %s", expectedResponse.OriginalPrompt, predResp.OriginalPrompt)
	}
	if len(predResp.Predictions) != len(expectedResponse.Predictions) {
		t.Errorf("Expected %d predictions, got %d", len(expectedResponse.Predictions), len(predResp.Predictions))
	}
	for i, pred := range predResp.Predictions {
		if pred.Timeframe != expectedResponse.Predictions[i].Timeframe {
			t.Errorf("Prediction %d: Expected timeframe %s, got %s", i, expectedResponse.Predictions[i].Timeframe, pred.Timeframe)
		}
		if pred.Description != expectedResponse.Predictions[i].Description {
			t.Errorf("Prediction %d: Expected description %s, got %s", i, expectedResponse.Predictions[i].Description, pred.Description)
		}
		if pred.Impact != expectedResponse.Predictions[i].Impact {
			t.Errorf("Prediction %d: Expected impact %s, got %s", i, expectedResponse.Predictions[i].Impact, pred.Impact)
		}
	}

	expectedBytes, err := json.Marshal(expectedResponse)
	if err != nil {
		t.Fatalf("Failed to marshal expected response: %v", err)
	}
	expectedCompact := string(expectedBytes)
	if result != expectedCompact {
		t.Errorf("Expected response: %s, got: %s", expectedCompact, result)
	}
}

func TestAlwaysInvalidResponse(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "testkey")
	originalDelay := retryDelay
	retryDelay = 1 * time.Millisecond
	defer func() { retryDelay = originalDelay }()

	client := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("invalid json")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, err := generatePredictions("test event", client)
	if err == nil {
		t.Error("Expected error after 10 invalid responses, got nil")
	}
}

func TestMismatchedOriginalPrompt(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "testkey")
	originalDelay := retryDelay
	retryDelay = 1 * time.Millisecond
	defer func() { retryDelay = originalDelay }()

	// Return a JSON where original_prompt does not match input
	resp := `{"original_prompt": "different event", "predictions": [{"timeframe": "1 week", "description": "Event A", "impact": "Market volatility"}]}`
	client := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(resp)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, err := generatePredictions("test event", client)
	if err == nil {
		t.Error("Expected error due to mismatched original_prompt, got nil")
	}
}

func TestEmptyPredictions(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "testkey")
	originalDelay := retryDelay
	retryDelay = 1 * time.Millisecond
	defer func() { retryDelay = originalDelay }()

	// Return JSON with an empty predictions array
	resp := `{"original_prompt": "test event", "predictions": []}`
	client := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(resp)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, err := generatePredictions("test event", client)
	if err == nil {
		t.Error("Expected error due to empty predictions array, got nil")
	}
}

func TestPredictionMissingField(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "testkey")
	originalDelay := retryDelay
	retryDelay = 1 * time.Millisecond
	defer func() { retryDelay = originalDelay }()

	// Return JSON with a prediction missing a required field (empty description)
	resp := `{"original_prompt": "test event", "predictions": [{"timeframe": "1 week", "description": "", "impact": "Market volatility"}]}`
	client := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(resp)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, err := generatePredictions("test event", client)
	if err == nil {
		t.Error("Expected error due to missing prediction field, got nil")
	}
}

func TestAPIReturnsHTTPError(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "testkey")
	originalDelay := retryDelay
	retryDelay = 1 * time.Millisecond
	defer func() { retryDelay = originalDelay }()

	// Return an HTTP error status, e.g., 500 Internal Server Error
	client := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBufferString("Internal Server Error")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, err := generatePredictions("test event", client)
	if err == nil {
		t.Error("Expected error due to HTTP error status from API, got nil")
	}
}

// New tests for the critique agent functionality

func TestCritiquedValidResponse(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "testkey")
	originalDelay := retryDelay
	retryDelay = 1 * time.Millisecond
	defer func() { retryDelay = originalDelay }()

	client := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			bodyBytes, _ := io.ReadAll(req.Body)
			bodyStr := string(bodyBytes)
			if strings.Contains(bodyStr, "predictor of future stock market events") {
				// Simulate first agent response
				validResp := `{"original_prompt": "test event", "predictions": [{"timeframe": "1 week", "description": "Event A", "impact": "Market volatility"}]}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(validResp)),
					Header:     make(http.Header),
				}, nil
			} else if strings.Contains(bodyStr, "Critically review") {
				// Simulate valid critique response
				critiquedResp := `{"original_prompt": "test event", "predictions": [{"timeframe": "1 week", "description": "Event A", "impact": "Market volatility", "confidence": 0.95, "critique": "Likely due to favorable conditions"}]}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(critiquedResp)),
					Header:     make(http.Header),
				}, nil
			}
			return nil, errors.New("unexpected request")
		}),
	}

	result, err := generateCritiquedPredictions("test event", client)
	if err != nil {
		t.Fatalf("Expected valid critique response, got error: %v", err)
	}
	var cr CritiquedResponse
	err = json.Unmarshal([]byte(result), &cr)
	if err != nil {
		t.Fatalf("Failed to parse critique response: %v", err)
	}
	if cr.OriginalPrompt != "test event" {
		t.Errorf("Expected original_prompt 'test event', got: %s", cr.OriginalPrompt)
	}
	if len(cr.Predictions) != 1 {
		t.Errorf("Expected 1 prediction, got: %d", len(cr.Predictions))
	}
	pred := cr.Predictions[0]
	if pred.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got: %f", pred.Confidence)
	}
	if pred.Critique != "Likely due to favorable conditions" {
		t.Errorf("Expected critique 'Likely due to favorable conditions', got: %s", pred.Critique)
	}
}

func TestCritiquedAPIFailure(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "testkey")
	originalDelay := retryDelay
	retryDelay = 1 * time.Millisecond
	defer func() { retryDelay = originalDelay }()

	client := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			bodyBytes, _ := io.ReadAll(req.Body)
			bodyStr := string(bodyBytes)
			if strings.Contains(bodyStr, "predictor of future stock market events") {
				// First agent returns valid predictions.
				validResp := `{"original_prompt": "test event", "predictions": [{"timeframe": "1 week", "description": "Event A", "impact": "Market volatility"}]}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(validResp)),
					Header:     make(http.Header),
				}, nil
			} else if strings.Contains(bodyStr, "Critically review") {
				// Second agent simulates an API failure.
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("Internal Server Error")),
					Header:     make(http.Header),
				}, nil
			}
			return nil, errors.New("unexpected request")
		}),
	}

	_, err := generateCritiquedPredictions("test event", client)
	if err == nil {
		t.Error("Expected error due to second agent API failure, got nil")
	}
}

func TestCritiquedInvalidJSONResponse(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "testkey")
	originalDelay := retryDelay
	retryDelay = 1 * time.Millisecond
	defer func() { retryDelay = originalDelay }()

	callCount := 0
	client := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			bodyBytes, _ := io.ReadAll(req.Body)
			bodyStr := string(bodyBytes)
			if strings.Contains(bodyStr, "predictor of future stock market events") {
				// Valid first agent response.
				validResp := `{"original_prompt": "test event", "predictions": [{"timeframe": "1 week", "description": "Event A", "impact": "Market volatility"}]}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(validResp)),
					Header:     make(http.Header),
				}, nil
			} else if strings.Contains(bodyStr, "Critically review") {
				callCount++
				// Always return invalid JSON for second agent.
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("invalid json")),
					Header:     make(http.Header),
				}, nil
			}
			return nil, errors.New("unexpected request")
		}),
	}

	_, err := generateCritiquedPredictions("test event", client)
	if err == nil {
		t.Error("Expected error due to invalid critique JSON, got nil")
	}
	if callCount < 10 {
		t.Errorf("Expected at least 10 attempts for critique call, got: %d", callCount)
	}
}
