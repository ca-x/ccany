package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ccany/internal/models"

	"github.com/sirupsen/logrus"
)

// GeminiClient handles communication with Google's native Gemini API
type GeminiClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
	logger  *logrus.Logger
}

// NewGeminiClient creates a new Gemini client for native API communication
func NewGeminiClient(apiKey, baseURL string, timeout int, logger *logrus.Logger) *GeminiClient {
	// Convert OpenAI-compatible URL to native Gemini API URL
	if strings.Contains(baseURL, "generativelanguage.googleapis.com") {
		// Remove OpenAI-specific paths and construct native URL
		baseURL = strings.ReplaceAll(baseURL, "/openai", "")
		baseURL = strings.ReplaceAll(baseURL, "/v1beta/openai", "/v1beta/models")
		baseURL = strings.TrimSuffix(baseURL, "/")

		// Ensure it ends with /models for the native API
		if !strings.HasSuffix(baseURL, "/models") {
			if strings.Contains(baseURL, "/v1beta") {
				baseURL = strings.Replace(baseURL, "/v1beta", "/v1beta/models", 1)
			} else {
				baseURL = baseURL + "/v1beta/models"
			}
		}
	} else {
		// Default to native Gemini API endpoint
		baseURL = "https://generativelanguage.googleapis.com/v1beta/models"
	}

	logger.WithField("native_gemini_url", baseURL).Info("🔧 Configured native Gemini API endpoint")

	return &GeminiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		logger: logger,
	}
}

// CreateChatCompletion sends a request to Gemini's native generateContent endpoint
func (c *GeminiClient) CreateChatCompletion(ctx context.Context, model string, req *models.GeminiRequest) (*models.GeminiResponse, error) {
	// Construct the native Gemini API URL
	// Format: https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent?key={apiKey}
	fullURL := fmt.Sprintf("%s/%s:generateContent?key=%s", c.baseURL, model, c.apiKey)

	// Marshal the request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini request: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"url":   fullURL,
		"model": model,
		"tools": len(req.Tools),
	}).Info("🔧 Sending request to Gemini Native API")

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini http request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send gemini request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read gemini response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"status_code": resp.StatusCode,
		"body_size":   len(body),
	}).Info("🔧 Received response from Gemini Native API")

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var errorResp models.GeminiErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("gemini api error (status %d): %s", resp.StatusCode, errorResp.Error.Message)
		}
		return nil, fmt.Errorf("gemini api error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse successful response
	var geminiResp models.GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gemini response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"candidates": len(geminiResp.Candidates),
		"usage":      geminiResp.UsageMetadata,
	}).Info("🔧 Successfully parsed Gemini response")

	return &geminiResp, nil
}

// CreateStreamingChatCompletion sends a streaming request to Gemini's native API
func (c *GeminiClient) CreateStreamingChatCompletion(ctx context.Context, model string, req *models.GeminiRequest) (*http.Response, error) {
	// Construct the native Gemini streaming API URL
	// Format: https://generativelanguage.googleapis.com/v1beta/models/{model}:streamGenerateContent?key={apiKey}
	fullURL := fmt.Sprintf("%s/%s:streamGenerateContent?key=%s", c.baseURL, model, c.apiKey)

	// Marshal the request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini streaming request: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"url":   fullURL,
		"model": model,
		"tools": len(req.Tools),
	}).Info("🔧 Sending streaming request to Gemini Native API")

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini streaming http request: %w", err)
	}

	// Set headers for streaming
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// Send the request and return the response for streaming processing
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send gemini streaming request: %w", err)
	}

	// Check for immediate HTTP errors
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		// Try to parse error response
		var errorResp models.GeminiErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("gemini streaming api error (status %d): %s", resp.StatusCode, errorResp.Error.Message)
		}
		return nil, fmt.Errorf("gemini streaming api error (status %d): %s", resp.StatusCode, string(body))
	}

	c.logger.Info("🔧 Gemini streaming connection established")
	return resp, nil
}

// ValidateAPIKey validates the Gemini API key by making a simple request
func (c *GeminiClient) ValidateAPIKey(ctx context.Context) error {
	// Create a minimal test request
	testReq := &models.GeminiRequest{
		Contents: []models.GeminiContent{
			{
				Role: "user",
				Parts: []models.GeminiPart{
					{Text: "Hello"},
				},
			},
		},
		GenerationConfig: &models.GeminiGenerationConfig{
			MaxOutputTokens: func(i int) *int { return &i }(1),
		},
	}

	// Use a basic model for validation
	_, err := c.CreateChatCompletion(ctx, "gemini-1.5-flash", testReq)
	return err
}

// GetModelInfo returns information about available Gemini models
func (c *GeminiClient) GetModelInfo(ctx context.Context) (map[string]any, error) {
	// This would typically call the models endpoint, but for now return static info
	return map[string]any{
		"models": []string{
			"gemini-1.5-flash",
			"gemini-1.5-flash-latest",
			"gemini-1.5-pro",
			"gemini-1.5-pro-latest",
			"gemini-2.5-flash",
		},
		"supports_tools":     true,
		"supports_streaming": true,
		"supports_vision":    true,
		"max_tokens":         8192,
		"context_window":     1000000,
	}, nil
}
