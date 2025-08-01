package handlers

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ccany/internal/client"
	"ccany/internal/config"
	"ccany/internal/converter"
	"ccany/internal/logging"
	"ccany/internal/models"
	"ccany/internal/session"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// MessagesHandler handles Claude messages API requests
type MessagesHandler struct {
	config         *config.Config
	openaiClient   *client.OpenAIClient
	requestLogger  *logging.RequestLogger
	logger         *logrus.Logger
	sessionManager *session.SessionManager
}

// NewMessagesHandler creates a new messages handler
func NewMessagesHandler(cfg *config.Config, openaiClient *client.OpenAIClient, requestLogger *logging.RequestLogger, logger *logrus.Logger, sessionManager *session.SessionManager) *MessagesHandler {
	return &MessagesHandler{
		config:         cfg,
		openaiClient:   openaiClient,
		requestLogger:  requestLogger,
		logger:         logger,
		sessionManager: sessionManager,
	}
}

// CreateMessage handles POST /v1/messages
func (h *MessagesHandler) CreateMessage(c *gin.Context) {
	var claudeReq models.ClaudeMessagesRequest
	if err := c.ShouldBindJSON(&claudeReq); err != nil {
		h.logger.WithError(err).Error("Failed to bind request")
		c.JSON(http.StatusBadRequest, converter.CreateClaudeErrorResponse("invalid_request_error", "Invalid request format"))
		return
	}

	requestID := uuid.New().String()

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"model":      claudeReq.Model,
		"stream":     claudeReq.Stream,
		"max_tokens": claudeReq.MaxTokens,
	}).Info("Processing Claude messages request")

	// Extract session context from headers or request
	projectPath := c.GetHeader("X-Claude-Project-Path")
	userID := c.GetHeader("X-Claude-User-ID")
	if projectPath == "" {
		projectPath = "default"
	}
	if userID == "" {
		userID = "anonymous"
	}

	// Get or create session for conversation context
	var allMessages []models.ClaudeMessage
	if h.sessionManager != nil {
		session, err := h.sessionManager.GetOrCreateSession(projectPath, userID, claudeReq.System)
		if err != nil {
			h.logger.WithError(err).Warn("Failed to get session, proceeding without context")
			allMessages = claudeReq.Messages
		} else {
			// Get previous messages from session
			sessionMessages, systemPrompt, err := h.sessionManager.GetSessionMessages(session.ID)
			if err != nil {
				h.logger.WithError(err).Warn("Failed to get session messages")
				allMessages = claudeReq.Messages
			} else {
				// Combine session messages with new message
				allMessages = append(sessionMessages, claudeReq.Messages...)

				// Use system prompt from session if not provided in request
				if claudeReq.System == nil && systemPrompt != nil {
					claudeReq.System = systemPrompt
				}

				h.logger.WithFields(logrus.Fields{
					"session_id":        session.ID,
					"previous_messages": len(sessionMessages),
					"new_messages":      len(claudeReq.Messages),
					"total_messages":    len(allMessages),
				}).Debug("Using conversation context from session")
			}
		}
	} else {
		allMessages = claudeReq.Messages
	}

	// Create modified request with full conversation history
	fullReq := claudeReq
	fullReq.Messages = allMessages

	// Convert Claude request to OpenAI format
	openaiReq, err := converter.ConvertClaudeToOpenAI(&fullReq, h.config.BigModel, h.config.SmallModel)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert request")
		c.JSON(http.StatusBadRequest, converter.CreateClaudeErrorResponse("invalid_request_error", "Failed to convert request"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.config.RequestTimeout)*time.Second)
	defer cancel()

	if claudeReq.Stream {
		h.handleStreamingRequest(c, ctx, requestID, &claudeReq, openaiReq, projectPath, userID)
	} else {
		h.handleNonStreamingRequest(c, ctx, requestID, &claudeReq, openaiReq, projectPath, userID)
	}
}

// handleNonStreamingRequest handles non-streaming requests
func (h *MessagesHandler) handleNonStreamingRequest(c *gin.Context, ctx context.Context, requestID string, claudeReq *models.ClaudeMessagesRequest, openaiReq *models.OpenAIChatCompletionRequest, projectPath, userID string) {
	startTime := time.Now()

	// Send request to OpenAI
	openaiResp, err := h.openaiClient.CreateChatCompletion(ctx, openaiReq)
	duration := time.Since(startTime)

	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"duration":   duration,
			"error":      err.Error(),
		}).Error("OpenAI request failed")

		// log failed request
		if h.requestLogger != nil {
			logData := &logging.RequestLogData{
				ID:           requestID,
				ClaudeModel:  claudeReq.Model,
				OpenAIModel:  openaiReq.Model,
				RequestBody:  claudeReq,
				ResponseBody: nil,
				StatusCode:   http.StatusInternalServerError,
				IsStreaming:  false,
				InputTokens:  0,
				OutputTokens: 0,
				DurationMs:   float64(duration.Milliseconds()),
				ErrorMessage: err.Error(),
				CreatedAt:    startTime,
			}
			go func() {
				if err := h.requestLogger.LogRequest(context.Background(), logData); err != nil {
					h.logger.WithError(err).Error("Failed to log request")
				}
			}()
		}

		errorType := h.openaiClient.ClassifyError(err)
		c.JSON(http.StatusInternalServerError, converter.CreateClaudeErrorResponse(errorType, err.Error()))
		return
	}

	// Convert OpenAI response to Claude format
	claudeResp, err := converter.ConvertOpenAIToClaudeResponse(openaiResp, claudeReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert response")

		// log conversion failed request
		if h.requestLogger != nil {
			logData := &logging.RequestLogData{
				ID:           requestID,
				ClaudeModel:  claudeReq.Model,
				OpenAIModel:  openaiReq.Model,
				RequestBody:  claudeReq,
				ResponseBody: openaiResp,
				StatusCode:   http.StatusInternalServerError,
				IsStreaming:  false,
				InputTokens:  0,
				OutputTokens: 0,
				DurationMs:   float64(duration.Milliseconds()),
				ErrorMessage: "Failed to convert response",
				CreatedAt:    startTime,
			}
			go func() {
				if err := h.requestLogger.LogRequest(context.Background(), logData); err != nil {
					h.logger.WithError(err).Error("Failed to log request")
				}
			}()
		}

		c.JSON(http.StatusInternalServerError, converter.CreateClaudeErrorResponse("api_error", "Failed to convert response"))
		return
	}

	h.logger.WithFields(logrus.Fields{
		"request_id":    requestID,
		"duration":      duration,
		"input_tokens":  claudeResp.Usage.InputTokens,
		"output_tokens": claudeResp.Usage.OutputTokens,
		"stop_reason":   claudeResp.StopReason,
	}).Info("Request completed successfully")

	// log successful request
	if h.requestLogger != nil {
		logData := &logging.RequestLogData{
			ID:           requestID,
			ClaudeModel:  claudeReq.Model,
			OpenAIModel:  openaiReq.Model,
			RequestBody:  claudeReq,
			ResponseBody: claudeResp,
			StatusCode:   http.StatusOK,
			IsStreaming:  false,
			InputTokens:  claudeResp.Usage.InputTokens,
			OutputTokens: claudeResp.Usage.OutputTokens,
			DurationMs:   float64(duration.Milliseconds()),
			ErrorMessage: "",
			CreatedAt:    startTime,
		}
		go func() {
			if err := h.requestLogger.LogRequest(context.Background(), logData); err != nil {
				h.logger.WithError(err).Error("Failed to log request")
			}
		}()
	}

	c.JSON(http.StatusOK, claudeResp)

	// Save messages to session for context
	if h.sessionManager != nil {
		sessionID := h.generateSessionID(projectPath, userID)

		// Add user messages to session
		for _, msg := range claudeReq.Messages {
			if err := h.sessionManager.AddMessage(sessionID, msg, claudeResp.Usage.InputTokens/len(claudeReq.Messages)); err != nil {
				h.logger.WithError(err).Warn("Failed to add user message to session")
			}
		}

		// Add assistant response to session
		assistantMsg := models.ClaudeMessage{
			Role:    "assistant",
			Content: claudeResp.Content,
		}
		if err := h.sessionManager.AddMessage(sessionID, assistantMsg, claudeResp.Usage.OutputTokens); err != nil {
			h.logger.WithError(err).Warn("Failed to add assistant message to session")
		}
	}
}

// handleStreamingRequest handles streaming requests
func (h *MessagesHandler) handleStreamingRequest(c *gin.Context, ctx context.Context, requestID string, claudeReq *models.ClaudeMessagesRequest, openaiReq *models.OpenAIChatCompletionRequest, projectPath, userID string) {
	startTime := time.Now()

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "*")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Get streaming response from OpenAI
	streamChan, err := h.openaiClient.CreateChatCompletionStream(ctx, openaiReq)
	if err != nil {
		duration := time.Since(startTime)
		h.logger.WithError(err).Error("Failed to create stream")

		// log streaming request failure
		if h.requestLogger != nil {
			logData := &logging.RequestLogData{
				ID:           requestID,
				ClaudeModel:  claudeReq.Model,
				OpenAIModel:  openaiReq.Model,
				RequestBody:  claudeReq,
				ResponseBody: nil,
				StatusCode:   http.StatusInternalServerError,
				IsStreaming:  true,
				InputTokens:  0,
				OutputTokens: 0,
				DurationMs:   float64(duration.Milliseconds()),
				ErrorMessage: err.Error(),
				CreatedAt:    startTime,
			}
			go func() {
				if err := h.requestLogger.LogRequest(context.Background(), logData); err != nil {
					h.logger.WithError(err).Error("Failed to log request")
				}
			}()
		}

		h.writeSSEError(c, converter.CreateClaudeErrorResponse(h.openaiClient.ClassifyError(err), err.Error()))
		return
	}

	// Send stream start event
	startEvent := converter.CreateClaudeStreamStartEvent(requestID, claudeReq.Model)
	h.writeSSEEvent(c, "message_start", startEvent)

	// Send ping event
	pingEvent := converter.CreateClaudeStreamPingEvent()
	h.writeSSEEvent(c, "ping", pingEvent)

	// Create streaming context for proper Claude format
	streamCtx := converter.CreateStreamingContext(requestID, claudeReq.Model, 0) // TODO: Calculate input tokens

	var totalInputTokens, totalOutputTokens int
	hasError := false
	var streamError error
	var responseContent strings.Builder

	// Process stream chunks
	for chunk := range streamChan {
		select {
		case <-ctx.Done():
			h.logger.WithField("request_id", requestID).Warn("Request context cancelled")
			hasError = true
			streamError = ctx.Err()
			goto exitLoop
		default:
		}

		if chunk.Error != nil {
			h.logger.WithError(chunk.Error).Error("Stream error")
			h.writeSSEError(c, converter.CreateClaudeErrorResponse("api_error", chunk.Error.Error()))
			hasError = true
			streamError = chunk.Error
			goto exitLoop
		}

		if chunk.Done {
			goto exitLoop
		}

		if chunk.Data != nil {
			// Convert OpenAI stream chunk to Claude events
			events, err := converter.ConvertOpenAIStreamToClaudeStream(chunk.Data, claudeReq, streamCtx)
			if err != nil {
				h.logger.WithError(err).Warn("Failed to convert stream chunk")
				continue
			}

			// Send each event
			for _, event := range events {
				h.writeSSEEvent(c, event.Type, event)

				// Collect content for session storage
				if event.Delta != nil && event.Delta.Text != "" {
					responseContent.WriteString(event.Delta.Text)
				}
			}

			// Update token counts from usage if available (Note: OpenAIStreamResponse doesn't have Usage)
			// Token counts would need to be calculated differently for streaming responses
			// For now, we'll rely on final usage reporting from the complete response
		}
	}

exitLoop:
	// Send final usage event if we have token counts
	if totalInputTokens > 0 || totalOutputTokens > 0 {
		usage := models.ClaudeUsage{
			InputTokens:  totalInputTokens,
			OutputTokens: totalOutputTokens,
		}
		stopEvent := converter.CreateClaudeStreamStopEvent(usage)
		h.writeSSEEvent(c, "message_delta", stopEvent)
	}

	// Send message_stop event
	stopEvent := models.ClaudeStreamEvent{Type: "message_stop"}
	h.writeSSEEvent(c, "message_stop", stopEvent)

	duration := time.Since(startTime)
	h.logger.WithFields(logrus.Fields{
		"request_id":    requestID,
		"duration":      duration,
		"input_tokens":  totalInputTokens,
		"output_tokens": totalOutputTokens,
	}).Info("Streaming request completed")

	// log streaming request result
	if h.requestLogger != nil {
		statusCode := http.StatusOK
		errorMessage := ""

		if hasError {
			statusCode = http.StatusInternalServerError
			if streamError != nil {
				errorMessage = streamError.Error()
			}
		}

		logData := &logging.RequestLogData{
			ID:          requestID,
			ClaudeModel: claudeReq.Model,
			OpenAIModel: openaiReq.Model,
			RequestBody: claudeReq,
			ResponseBody: gin.H{
				"usage":     models.ClaudeUsage{InputTokens: totalInputTokens, OutputTokens: totalOutputTokens},
				"streaming": true,
			},
			StatusCode:   statusCode,
			IsStreaming:  true,
			InputTokens:  totalInputTokens,
			OutputTokens: totalOutputTokens,
			DurationMs:   float64(duration.Milliseconds()),
			ErrorMessage: errorMessage,
			CreatedAt:    startTime,
		}
		go func() {
			if err := h.requestLogger.LogRequest(context.Background(), logData); err != nil {
				h.logger.WithError(err).Error("Failed to log request")
			}
		}()
	}

	// Save messages to session for context (only if successful)
	if !hasError && h.sessionManager != nil {
		sessionID := h.generateSessionID(projectPath, userID)

		// Add user messages to session
		for _, msg := range claudeReq.Messages {
			if err := h.sessionManager.AddMessage(sessionID, msg, totalInputTokens/len(claudeReq.Messages)); err != nil {
				h.logger.WithError(err).Warn("Failed to add user message to session")
			}
		}

		// Add assistant response to session if we have content
		if responseContent.Len() > 0 {
			assistantMsg := models.ClaudeMessage{
				Role:    "assistant",
				Content: responseContent.String(),
			}
			if err := h.sessionManager.AddMessage(sessionID, assistantMsg, totalOutputTokens); err != nil {
				h.logger.WithError(err).Warn("Failed to add assistant message to session")
			}
		}
	}
}

// writeSSEEvent writes an SSE event to the response
func (h *MessagesHandler) writeSSEEvent(c *gin.Context, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal SSE event")
		return
	}

	if _, err := fmt.Fprintf(c.Writer, "event: %s\n", eventType); err != nil {
		h.logger.WithError(err).Error("Failed to write SSE event type")
		return
	}
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonData)); err != nil {
		h.logger.WithError(err).Error("Failed to write SSE data")
		return
	}
	c.Writer.Flush()
}

// writeSSEError writes an SSE error event
func (h *MessagesHandler) writeSSEError(c *gin.Context, errorResp *models.ClaudeErrorResponse) {
	h.writeSSEEvent(c, "error", errorResp)
}

// CountTokens handles POST /v1/messages/count_tokens
func (h *MessagesHandler) CountTokens(c *gin.Context) {
	var req models.ClaudeTokenCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, converter.CreateClaudeErrorResponse("invalid_request_error", "Invalid request format"))
		return
	}

	// Use enhanced token counter from EnhancedMessagesHandler if available
	enhancedHandler := NewEnhancedMessagesHandler(h.config, nil, h.openaiClient, h.requestLogger, h.logger)

	// Convert to ClaudeMessagesRequest for token counting
	claudeReq := &models.ClaudeMessagesRequest{
		Model:    req.Model,
		Messages: req.Messages,
		System:   req.System,
	}

	estimatedTokens := enhancedHandler.EstimateTokenCount(claudeReq)

	h.logger.WithFields(logrus.Fields{
		"model":            req.Model,
		"estimated_tokens": estimatedTokens,
		"message_count":    len(req.Messages),
	}).Info("Token count estimation completed")

	c.JSON(http.StatusOK, gin.H{
		"input_tokens": estimatedTokens,
	})
}

// CreateChatCompletion handles POST /v1/chat/completions (OpenAI API compatible)
func (h *MessagesHandler) CreateChatCompletion(c *gin.Context) {
	var openaiReq models.OpenAIChatCompletionRequest
	if err := c.ShouldBindJSON(&openaiReq); err != nil {
		h.logger.WithError(err).Error("Failed to bind OpenAI request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request format",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	requestID := uuid.New().String()

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"model":      openaiReq.Model,
		"stream":     openaiReq.Stream,
		"max_tokens": openaiReq.MaxTokens,
	}).Info("Processing OpenAI chat completion request")

	// use model mapping from configuration
	switch openaiReq.Model {
	case "claude-3-5-sonnet-20241022", "claude-3-haiku-20240307":
		openaiReq.Model = h.config.BigModel
	default:
		openaiReq.Model = h.config.SmallModel
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.config.RequestTimeout)*time.Second)
	defer cancel()

	if openaiReq.Stream {
		h.handleOpenAIStreamingRequest(c, ctx, requestID, &openaiReq)
	} else {
		h.handleOpenAINonStreamingRequest(c, ctx, requestID, &openaiReq)
	}
}

// handleOpenAINonStreamingRequest handles non-streaming OpenAI requests
func (h *MessagesHandler) handleOpenAINonStreamingRequest(c *gin.Context, ctx context.Context, requestID string, openaiReq *models.OpenAIChatCompletionRequest) {
	startTime := time.Now()

	// Send request to OpenAI
	openaiResp, err := h.openaiClient.CreateChatCompletion(ctx, openaiReq)
	duration := time.Since(startTime)

	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"duration":   duration,
			"error":      err.Error(),
		}).Error("OpenAI request failed")

		// log failed request
		if h.requestLogger != nil {
			logData := &logging.RequestLogData{
				ID:           requestID,
				ClaudeModel:  "openai-direct",
				OpenAIModel:  openaiReq.Model,
				RequestBody:  openaiReq,
				ResponseBody: nil,
				StatusCode:   http.StatusInternalServerError,
				IsStreaming:  false,
				InputTokens:  0,
				OutputTokens: 0,
				DurationMs:   float64(duration.Milliseconds()),
				ErrorMessage: err.Error(),
				CreatedAt:    startTime,
			}
			go func() {
				if err := h.requestLogger.LogRequest(context.Background(), logData); err != nil {
					h.logger.WithError(err).Error("Failed to log request")
				}
			}()
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "api_error",
			},
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"duration":   duration,
	}).Info("OpenAI request completed successfully")

	// log successful request
	if h.requestLogger != nil {
		logData := &logging.RequestLogData{
			ID:           requestID,
			ClaudeModel:  "openai-direct",
			OpenAIModel:  openaiReq.Model,
			RequestBody:  openaiReq,
			ResponseBody: openaiResp,
			StatusCode:   http.StatusOK,
			IsStreaming:  false,
			InputTokens:  0,
			OutputTokens: 0,
			DurationMs:   float64(duration.Milliseconds()),
			ErrorMessage: "",
			CreatedAt:    startTime,
		}
		go func() {
			if err := h.requestLogger.LogRequest(context.Background(), logData); err != nil {
				h.logger.WithError(err).Error("Failed to log request")
			}
		}()
	}

	c.JSON(http.StatusOK, openaiResp)
}

// handleOpenAIStreamingRequest handles streaming OpenAI requests
func (h *MessagesHandler) handleOpenAIStreamingRequest(c *gin.Context, ctx context.Context, requestID string, openaiReq *models.OpenAIChatCompletionRequest) {
	startTime := time.Now()

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "*")

	// Get streaming response from OpenAI
	streamChan, err := h.openaiClient.CreateChatCompletionStream(ctx, openaiReq)
	if err != nil {
		duration := time.Since(startTime)
		h.logger.WithError(err).Error("Failed to create OpenAI stream")

		// log streaming request failure
		if h.requestLogger != nil {
			logData := &logging.RequestLogData{
				ID:           requestID,
				ClaudeModel:  "openai-direct",
				OpenAIModel:  openaiReq.Model,
				RequestBody:  openaiReq,
				ResponseBody: nil,
				StatusCode:   http.StatusInternalServerError,
				IsStreaming:  true,
				InputTokens:  0,
				OutputTokens: 0,
				DurationMs:   float64(duration.Milliseconds()),
				ErrorMessage: err.Error(),
				CreatedAt:    startTime,
			}
			go func() {
				if err := h.requestLogger.LogRequest(context.Background(), logData); err != nil {
					h.logger.WithError(err).Error("Failed to log request")
				}
			}()
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "api_error",
			},
		})
		return
	}

	var totalInputTokens, totalOutputTokens int
	hasError := false
	var streamError error

	// Process stream chunks
	for chunk := range streamChan {
		select {
		case <-ctx.Done():
			h.logger.WithField("request_id", requestID).Warn("Request context cancelled")
			hasError = true
			streamError = ctx.Err()
			goto exitOpenAILoop
		default:
		}

		if chunk.Error != nil {
			h.logger.WithError(chunk.Error).Error("Stream error")
			hasError = true
			streamError = chunk.Error
			goto exitOpenAILoop
		}

		if chunk.Done {
			goto exitOpenAILoop
		}

		if chunk.Data != nil {
			// directly output OpenAI format streaming response
			jsonData, err := json.Marshal(chunk.Data)
			if err != nil {
				h.logger.WithError(err).Error("Failed to marshal stream chunk")
				continue
			}

			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonData)); err != nil {
				h.logger.WithError(err).Error("Failed to write stream data")
				continue
			}
			c.Writer.Flush()
		}
	}

exitOpenAILoop:
	// Send [DONE] signal
	if _, err := fmt.Fprintf(c.Writer, "data: [DONE]\n\n"); err != nil {
		h.logger.WithError(err).Error("Failed to write [DONE] signal")
	}
	c.Writer.Flush()

	duration := time.Since(startTime)
	h.logger.WithFields(logrus.Fields{
		"request_id":    requestID,
		"duration":      duration,
		"input_tokens":  totalInputTokens,
		"output_tokens": totalOutputTokens,
	}).Info("OpenAI streaming request completed")

	// log streaming request result
	if h.requestLogger != nil {
		statusCode := http.StatusOK
		errorMessage := ""

		if hasError {
			statusCode = http.StatusInternalServerError
			if streamError != nil {
				errorMessage = streamError.Error()
			}
		}

		logData := &logging.RequestLogData{
			ID:          requestID,
			ClaudeModel: "openai-direct",
			OpenAIModel: openaiReq.Model,
			RequestBody: openaiReq,
			ResponseBody: gin.H{
				"streaming": true,
			},
			StatusCode:   statusCode,
			IsStreaming:  true,
			InputTokens:  totalInputTokens,
			OutputTokens: totalOutputTokens,
			DurationMs:   float64(duration.Milliseconds()),
			ErrorMessage: errorMessage,
			CreatedAt:    startTime,
		}
		go func() {
			if err := h.requestLogger.LogRequest(context.Background(), logData); err != nil {
				h.logger.WithError(err).Error("Failed to log request")
			}
		}()
	}
}

// generateSessionID generates a session ID based on project path and user ID
func (h *MessagesHandler) generateSessionID(projectPath, userID string) string {
	data := fmt.Sprintf("%s:%s", projectPath, userID)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("session_%x", hash)
}
