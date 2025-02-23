package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

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
				Body:       ioutil.NopCloser(bytes.NewBufferString(validResp)),
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
					Body:       ioutil.NopCloser(bytes.NewBufferString("invalid json")),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBufferString(validResp)),
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
				Body:       ioutil.NopCloser(bytes.NewBufferString("invalid json")),
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
				Body:       ioutil.NopCloser(bytes.NewBufferString(resp)),
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

	// Return JSON with empty predictions array
	resp := `{"original_prompt": "test event", "predictions": []}`
	client := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBufferString(resp)),
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
				Body:       ioutil.NopCloser(bytes.NewBufferString(resp)),
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

	// Return HTTP error status, e.g., 500 Internal Server Error
	client := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       ioutil.NopCloser(bytes.NewBufferString("Internal Server Error")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, err := generatePredictions("test event", client)
	if err == nil {
		t.Error("Expected error due to HTTP error status from API, got nil")
	}
}
