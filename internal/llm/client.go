package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Client represents an LLM API client
type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

// NewClient creates a new LLM API client
func NewClient(httpClient *http.Client) (*Client, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY is not set")
	}

	return &Client{
		httpClient: httpClient,
		apiKey:     apiKey,
		baseURL:    "https://api.openai.com/v1/chat/completions",
	}, nil
}

// CallLLM sends a request to the LLM API and returns the response
func (c *Client) CallLLM(prompt string) (string, error) {
	requestPayload := map[string]interface{}{
		"model": "o1-mini",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	
	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
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

	return c.parseResponse(bodyBytes)
}

func (c *Client) parseResponse(bodyBytes []byte) (string, error) {
	// Try parsing as LLM chat response first
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
	err := json.Unmarshal(bodyBytes, &lr)
	if err == nil && len(lr.Choices) > 0 {
		return sanitizeResponse(lr.Choices[0].Message.Content), nil
	}

	// Return raw response if can't parse as chat response
	return string(bodyBytes), nil
}

// sanitizeResponse cleans up the response string
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
