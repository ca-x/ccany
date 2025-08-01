package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ccany/internal/channel"
	"ccany/internal/converter"
)

// UnifiedAPIHandler handles unified API requests with auto-detection and conversion
type UnifiedAPIHandler struct {
	channelManager   *channel.ChannelManager
	formatDetector   *converter.FormatDetector
	converterFactory *converter.ConverterFactory
	logger           *logrus.Logger
}

// NewUnifiedAPIHandler creates a new unified API handler
func NewUnifiedAPIHandler(cm *channel.ChannelManager, logger *logrus.Logger) *UnifiedAPIHandler {
	return &UnifiedAPIHandler{
		channelManager:   cm,
		formatDetector:   converter.NewFormatDetector(),
		converterFactory: converter.NewConverterFactory(),
		logger:           logger,
	}
}

// UnifiedRequest represents a unified API request
type UnifiedRequest struct {
	SourceFormat converter.APIFormat `json:"source_format,omitempty"`
	TargetFormat converter.APIFormat `json:"target_format,omitempty"`
	ChannelID    string              `json:"channel_id,omitempty"`
	CustomKey    string              `json:"custom_key,omitempty"`
	AutoDetect   bool                `json:"auto_detect,omitempty"`
	Data         json.RawMessage     `json:"data"`
	Headers      map[string]string   `json:"headers,omitempty"`
}

// HandleUnifiedRequest handles requests with format auto-detection and conversion
func (h *UnifiedAPIHandler) HandleUnifiedRequest(c *gin.Context) {
	startTime := time.Now()

	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.WithError(err).Error("Failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Restore body for potential re-reading
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// Detect source format if not specified
	var sourceFormat converter.APIFormat
	var targetFormat converter.APIFormat
	var channelID string
	var customKey string

	// Try to parse as unified request first
	var unifiedReq UnifiedRequest
	if err := json.Unmarshal(body, &unifiedReq); err == nil && len(unifiedReq.Data) > 0 {
		// This is a unified request
		sourceFormat = unifiedReq.SourceFormat
		targetFormat = unifiedReq.TargetFormat
		channelID = unifiedReq.ChannelID
		customKey = unifiedReq.CustomKey
		body = unifiedReq.Data
	} else {
		// This is a direct API request, detect format
		detection := h.formatDetector.DetectFromRequest(c.Request, body)
		sourceFormat = detection.Format

		// Extract channel info from headers or query params
		customKey = h.extractCustomKey(c)
	}

	// Auto-detect source format if not specified
	if sourceFormat == converter.FormatUnknown {
		detection := h.formatDetector.DetectFromRequest(c.Request, body)
		sourceFormat = detection.Format

		h.logger.WithFields(logrus.Fields{
			"detected_format": sourceFormat,
			"confidence":      detection.Confidence,
			"reasons":         detection.Reasons,
		}).Info("Auto-detected API format")
	}

	if sourceFormat == converter.FormatUnknown {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unable to detect API format",
			"hint":  "Please specify source_format or ensure request follows a supported API format",
		})
		return
	}

	// Find target channel
	var targetChannel *channel.Channel
	if customKey != "" {
		if ch, exists := h.channelManager.GetChannelByCustomKey(customKey); exists {
			targetChannel = ch
			targetFormat = converter.APIFormat(ch.Provider)
		}
	} else if channelID != "" {
		if ch, exists := h.channelManager.GetChannel(channelID); exists {
			targetChannel = ch
			targetFormat = converter.APIFormat(ch.Provider)
		}
	}

	// If no specific channel, use source format provider
	if targetChannel == nil {
		var err error
		targetChannel, err = h.channelManager.RouteRequest(string(sourceFormat), channelID)
		if err != nil {
			h.logger.WithError(err).Error("Failed to route request")
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "No available channels for provider: " + string(sourceFormat),
			})
			return
		}
		targetFormat = converter.APIFormat(targetChannel.Provider)
	}

	// Convert request if needed
	var requestData map[string]interface{}
	if err := json.Unmarshal(body, &requestData); err != nil {
		h.logger.WithError(err).Error("Failed to parse request JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON in request body"})
		return
	}

	// Convert request format if needed
	if sourceFormat != targetFormat {
		converter, err := h.converterFactory.GetConverter(string(sourceFormat))
		if err != nil {
			h.logger.WithError(err).Error("Failed to get converter")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Conversion not supported"})
			return
		}

		convertedReq, err := converter.ConvertRequest(requestData, string(targetFormat), nil)
		if err != nil {
			h.logger.WithError(err).Error("Failed to convert request")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Request conversion failed: " + err.Error()})
			return
		}

		requestData = convertedReq.Data
	}

	// Forward request to target channel
	response, responseTime, tokenCount, err := h.forwardRequest(c.Request.Context(), targetChannel, requestData, c.Request.Header)

	// Record metrics
	success := err == nil
	if metricsErr := h.channelManager.UpdateChannelMetrics(c.Request.Context(), targetChannel.ID, responseTime, tokenCount, success); metricsErr != nil {
		h.logger.WithError(metricsErr).Warn("Failed to update channel metrics")
	}

	if err != nil {
		h.logger.WithError(err).Error("Failed to forward request")
		c.JSON(http.StatusBadGateway, gin.H{"error": "Upstream request failed: " + err.Error()})
		return
	}

	// Convert response if needed
	if sourceFormat != targetFormat {
		convertedResp, err := h.converterFactory.ConvertResponse(string(targetFormat), string(sourceFormat), response)
		if err != nil {
			h.logger.WithError(err).Error("Failed to convert response")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Response conversion failed: " + err.Error()})
			return
		}
		response = convertedResp.Data
	}

	// Add conversion metadata
	response["_conversion_info"] = map[string]interface{}{
		"source_format": sourceFormat,
		"target_format": targetFormat,
		"channel_id":    targetChannel.ID,
		"channel_name":  targetChannel.Name,
		"response_time": responseTime,
		"total_time":    time.Since(startTime).Seconds(),
	}

	c.JSON(http.StatusOK, response)
}

// HandleStreamingRequest handles streaming requests with format conversion
func (h *UnifiedAPIHandler) HandleStreamingRequest(c *gin.Context) {
	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.WithError(err).Error("Failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Auto-detect source format
	detection := h.formatDetector.DetectFromRequest(c.Request, body)
	sourceFormat := detection.Format

	if sourceFormat == converter.FormatUnknown {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unable to detect API format for streaming",
		})
		return
	}

	// Extract channel info
	customKey := h.extractCustomKey(c)

	// Find target channel
	var targetChannel *channel.Channel
	if customKey != "" {
		if ch, exists := h.channelManager.GetChannelByCustomKey(customKey); exists {
			targetChannel = ch
		}
	}

	if targetChannel == nil {
		var err error
		targetChannel, err = h.channelManager.RouteRequest(string(sourceFormat), "")
		if err != nil {
			h.logger.WithError(err).Error("Failed to route streaming request")
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "No available channels for streaming",
			})
			return
		}
	}

	targetFormat := converter.APIFormat(targetChannel.Provider)

	// Set up streaming response
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Forward streaming request
	h.forwardStreamingRequest(c, targetChannel, body, sourceFormat, targetFormat)
}

// extractCustomKey extracts custom key from various sources
func (h *UnifiedAPIHandler) extractCustomKey(c *gin.Context) string {
	// Try Authorization header
	if auth := c.GetHeader("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
	}

	// Try x-api-key header
	if apiKey := c.GetHeader("x-api-key"); apiKey != "" {
		return apiKey
	}

	// Try anthropic-specific header
	if apiKey := c.GetHeader("x-api-key"); apiKey != "" {
		return apiKey
	}

	// Try query parameter
	if apiKey := c.Query("api_key"); apiKey != "" {
		return apiKey
	}

	return ""
}

// forwardRequest forwards a request to the target channel
func (h *UnifiedAPIHandler) forwardRequest(ctx context.Context, targetChannel *channel.Channel, requestData map[string]interface{}, headers http.Header) (map[string]interface{}, float64, int64, error) {
	startTime := time.Now()

	// Prepare request
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", targetChannel.BaseURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+targetChannel.APIKey)

	// Copy relevant headers from original request
	for key, values := range headers {
		if h.shouldCopyHeader(key) {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	// Make request
	client := &http.Client{
		Timeout: time.Duration(targetChannel.Timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, time.Since(startTime).Seconds(), 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	responseTime := time.Since(startTime).Seconds()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, responseTime, 0, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, responseTime, 0, fmt.Errorf("upstream error %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse response
	var responseData map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		return nil, responseTime, 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// Estimate token count (simplified)
	tokenCount := h.estimateTokenCount(responseData)

	return responseData, responseTime, tokenCount, nil
}

// forwardStreamingRequest forwards a streaming request
func (h *UnifiedAPIHandler) forwardStreamingRequest(c *gin.Context, targetChannel *channel.Channel, requestBody []byte, _ /* sourceFormat */, _ /* targetFormat */ converter.APIFormat) {
	// This is a simplified implementation
	// In production, you'd want to handle actual streaming with proper SSE parsing and conversion

	ctx := c.Request.Context()

	// Create streaming request
	req, err := http.NewRequestWithContext(ctx, "POST", targetChannel.BaseURL, bytes.NewBuffer(requestBody))
	if err != nil {
		h.logger.WithError(err).Error("Failed to create streaming request")
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+targetChannel.APIKey)
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{
		Timeout: time.Duration(targetChannel.Timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		h.logger.WithError(err).Error("Streaming request failed")
		return
	}
	defer resp.Body.Close()

	// Stream response (simplified - in production you'd parse SSE events)
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		h.logger.WithError(err).Error("Failed to stream response")
	}
}

// shouldCopyHeader determines if a header should be copied to the upstream request
func (h *UnifiedAPIHandler) shouldCopyHeader(key string) bool {
	lowerKey := strings.ToLower(key)

	// Headers to copy
	allowedHeaders := map[string]bool{
		"user-agent":      true,
		"accept":          true,
		"accept-encoding": true,
		"accept-language": true,
	}

	// Headers to skip
	skipHeaders := map[string]bool{
		"host":           true,
		"authorization":  true,
		"content-length": true,
		"connection":     true,
	}

	if skipHeaders[lowerKey] {
		return false
	}

	return allowedHeaders[lowerKey]
}

// estimateTokenCount provides a rough estimate of token count from response
func (h *UnifiedAPIHandler) estimateTokenCount(responseData map[string]interface{}) int64 {
	// This is a very simplified estimation
	// In production, you'd want more accurate token counting based on the specific model

	if usage, ok := responseData["usage"].(map[string]interface{}); ok {
		if totalTokens, ok := usage["total_tokens"].(float64); ok {
			return int64(totalTokens)
		}
		if completionTokens, ok := usage["completion_tokens"].(float64); ok {
			return int64(completionTokens)
		}
	}

	// Fallback: estimate based on content length
	responseText, _ := json.Marshal(responseData)
	return int64(len(responseText) / 4) // Rough estimate: 4 chars per token
}

// RegisterUnifiedAPIRoutes registers unified API routes
func RegisterUnifiedAPIRoutes(router *gin.Engine, handler *UnifiedAPIHandler) {
	api := router.Group("/api/v1")
	{
		// Unified endpoints
		api.POST("/unified/chat", handler.HandleUnifiedRequest)
		api.POST("/unified/completion", handler.HandleUnifiedRequest)
		api.POST("/unified/stream", handler.HandleStreamingRequest)

		// Format-specific endpoints that auto-detect and convert
		api.POST("/openai/chat/completions", handler.HandleUnifiedRequest)
		api.POST("/v1/chat/completions", handler.HandleUnifiedRequest)
		api.POST("/anthropic/messages", handler.HandleUnifiedRequest)
		api.POST("/v1/messages", handler.HandleUnifiedRequest)
		api.POST("/gemini/generateContent", handler.HandleUnifiedRequest)
		api.POST("/v1beta/generateContent", handler.HandleUnifiedRequest)
	}
}
