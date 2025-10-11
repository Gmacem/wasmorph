package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPClient() *HTTPClient {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &HTTPClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (c *HTTPClient) CreateRule(apiKey, name, code string) (*http.Response, error) {
	payload := map[string]string{
		"name": name,
		"code": code,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/rules", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	return c.client.Do(req)
}

func (c *HTTPClient) ListRules(apiKey string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/v1/rules", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	return c.client.Do(req)
}

func (c *HTTPClient) ExecuteRule(apiKey, ruleName string, input map[string]any) (*http.Response, error) {
	jsonData, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/rules/"+ruleName+"/execute", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	return c.client.Do(req)
}

func (c *HTTPClient) DeleteRule(apiKey, ruleName string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", c.baseURL+"/api/v1/rules/"+ruleName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	return c.client.Do(req)
}

func (c *HTTPClient) HealthCheck() (*http.Response, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return c.client.Do(req)
}
