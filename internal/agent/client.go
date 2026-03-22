package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client calls an Anthropic-compatible Messages API.
type Client struct {
	provider   Provider
	httpClient *http.Client
}

// NewClient creates a new agent client for the given provider.
func NewClient(p Provider) *Client {
	return &Client{
		provider: p,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// message types for Anthropic Messages API

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model     string    `json:"model"`
	Messages  []message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type response struct {
	Content []contentBlock `json:"content"`
	Error   *apiError      `json:"error,omitempty"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Chat sends a message and returns the assistant's text response.
func (c *Client) Chat(system, userMsg string) (string, error) {
	reqBody := request{
		Model:     c.provider.Model,
		Messages:  []message{{Role: "user", Content: userMsg}},
		MaxTokens: 2048,
		System:    system,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("agent: marshal request: %w", err)
	}

	url := c.provider.BaseURL + "/v1/messages"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("agent: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.provider.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("agent: http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("agent: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("agent: API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("agent: parse response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("agent: %s: %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	// Extract text from content blocks
	var text string
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	if text == "" {
		return "", fmt.Errorf("agent: empty response")
	}
	return text, nil
}
