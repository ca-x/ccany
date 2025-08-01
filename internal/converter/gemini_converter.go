package converter

import (
	"ccany/internal/models"
	"fmt"
	"log"
	"time"
)

// GeminiConverter handles conversions involving Gemini format
type GeminiConverter struct{}

// NewGeminiConverter creates a new Gemini converter
func NewGeminiConverter() *GeminiConverter {
	return &GeminiConverter{}
}

// ConvertFromClaude converts Claude format request to Gemini format with robust schema sanitation.
func (c *GeminiConverter) ConvertFromClaude(claudeReq *models.ClaudeMessagesRequest) (*models.GeminiRequest, error) {
	geminiReq := &models.GeminiRequest{
		Contents: []models.GeminiContent{},
	}

	// 1. Handle System Prompt
	if claudeReq.System != nil {
		if systemStr, ok := claudeReq.System.(string); ok && systemStr != "" {
			geminiReq.SystemInstruction = &models.GeminiContent{
				Parts: []models.GeminiPart{{Text: systemStr}},
			}
		}
	}

	// 2. Handle Messages (This logic is correct and remains)
	for _, msg := range claudeReq.Messages {
		role := c.mapClaudeRoleToGemini(msg.Role)
		var parts []models.GeminiPart

		// Handle different content types
		switch content := msg.Content.(type) {
		case string:
			parts = append(parts, models.GeminiPart{Text: content})
		case []interface{}:
			for _, block := range content {
				blockMap := block.(map[string]interface{})
				switch blockMap["type"] {
				case "text":
					parts = append(parts, models.GeminiPart{Text: blockMap["text"].(string)})
				case "tool_use":
					parts = append(parts, models.GeminiPart{
						FunctionCall: &models.GeminiFunctionCall{
							Name: blockMap["name"].(string),
							Args: blockMap["input"].(map[string]interface{}),
						},
					})
				case "tool_result":
					// A tool result becomes its own "function" role message
					geminiReq.Contents = append(geminiReq.Contents, models.GeminiContent{
						Role: "function",
						Parts: []models.GeminiPart{{
							FunctionResponse: &models.GeminiFunctionResponse{
								Name:     blockMap["tool_use_id"].(string),
								Response: map[string]interface{}{"content": blockMap["content"]},
							},
						}},
					})
				}
			}
		}

		if len(parts) > 0 {
			geminiReq.Contents = append(geminiReq.Contents, models.GeminiContent{Role: role, Parts: parts})
		}
	}

	// 3. Handle Generation Config (This logic is correct and remains)
	config := &models.GeminiGenerationConfig{}
	if claudeReq.Temperature != nil {
		config.Temperature = claudeReq.Temperature
	}
	if claudeReq.MaxTokens > 0 {
		config.MaxOutputTokens = &claudeReq.MaxTokens
	}
	if claudeReq.TopP != nil {
		config.TopP = claudeReq.TopP
	}
	if claudeReq.TopK != nil {
		config.TopK = claudeReq.TopK
	}
	if len(claudeReq.StopSequences) > 0 {
		config.StopSequences = claudeReq.StopSequences
	}
	geminiReq.GenerationConfig = config

	// --- 4. REVISED AND CORRECTED TOOL CONVERSION ---
	if len(claudeReq.Tools) > 0 {
		var declarations []models.GeminiFunctionDeclaration

		for _, tool := range claudeReq.Tools {
			// Sanitize the entire schema recursively before creating the declaration.
			sanitizedSchema := c.sanitizeSchema(tool.InputSchema)

			// Convert sanitized schema to GeminiFunctionParameters
			var params *models.GeminiFunctionParameters
			if sanitizedSchema != nil {
				params = &models.GeminiFunctionParameters{}

				// Extract type (default to "object")
				if schemaType, ok := sanitizedSchema["type"].(string); ok {
					params.Type = schemaType
				} else {
					params.Type = "object"
				}

				// Extract properties
				if properties, ok := sanitizedSchema["properties"].(map[string]interface{}); ok {
					params.Properties = properties
				}

				// Extract required fields
				if required, ok := sanitizedSchema["required"].([]interface{}); ok {
					reqStrings := make([]string, len(required))
					for i, r := range required {
						if str, ok := r.(string); ok {
							reqStrings[i] = str
						}
					}
					params.Required = reqStrings
				}
			}

			declarations = append(declarations, models.GeminiFunctionDeclaration{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
			})
		}
		geminiReq.Tools = []models.GeminiTool{{FunctionDeclarations: declarations}}

		// The ToolConfig logic is still correct and necessary.
		if claudeReq.ToolChoice != nil {
			toolConfig := &models.GeminiToolConfig{
				FunctionCallingConfig: &models.GeminiFunctionCallingConfig{},
			}
			if tc, ok := claudeReq.ToolChoice.(string); ok {
				switch tc {
				case "auto":
					toolConfig.FunctionCallingConfig.Mode = "AUTO"
				case "required":
					toolConfig.FunctionCallingConfig.Mode = "ANY"
				case "none":
					toolConfig.FunctionCallingConfig.Mode = "NONE"
				}
			}
			geminiReq.ToolConfig = toolConfig
		} else {
			geminiReq.ToolConfig = &models.GeminiToolConfig{
				FunctionCallingConfig: &models.GeminiFunctionCallingConfig{Mode: "AUTO"},
			}
		}
	}

	// Debug logging
	if len(claudeReq.Tools) > 0 {
		log.Printf("🔧 DEBUG: Converted %d tools for Gemini.", len(geminiReq.Tools[0].FunctionDeclarations))
	}

	return geminiReq, nil
}

// sanitizeSchema recursively rebuilds a JSON Schema, keeping only Gemini-compatible fields.
func (c *GeminiConverter) sanitizeSchema(schema map[string]interface{}) map[string]interface{} {
	if schema == nil {
		return nil
	}

	sanitized := make(map[string]interface{})

	// Whitelist of allowed keys in a Gemini schema object.
	allowedKeys := map[string]bool{
		"type":        true,
		"description": true,
		"properties":  true,
		"required":    true,
		"items":       true,
		"enum":        true,
	}

	for key, value := range schema {
		if !allowedKeys[key] {
			continue // Skip unsupported keys like '$schema', 'additionalProperties', etc.
		}

		switch key {
		case "properties":
			if properties, ok := value.(map[string]interface{}); ok {
				sanitizedProps := make(map[string]interface{})
				for propName, propValue := range properties {
					if propSchema, ok := propValue.(map[string]interface{}); ok {
						// Recursively sanitize each individual property's schema.
						sanitizedProps[propName] = c.sanitizeSchema(propSchema)
					}
				}
				sanitized[key] = sanitizedProps
			}
		case "items":
			if items, ok := value.(map[string]interface{}); ok {
				// Recursively sanitize the 'items' schema for arrays.
				sanitized[key] = c.sanitizeSchema(items)
			}
		case "type":
			// Gemini requires a single type string, not an array of types.
			if typeStr, ok := value.(string); ok {
				sanitized[key] = typeStr
			} else if typeArray, ok := value.([]interface{}); ok && len(typeArray) > 0 {
				if firstType, ok := typeArray[0].(string); ok {
					sanitized[key] = firstType
				}
			}
		default:
			// Copy other allowed keys directly.
			sanitized[key] = value
		}
	}

	// Gemini requires a 'type' field for a schema to be valid.
	if _, exists := sanitized["type"]; !exists {
		// If no type is present, but there are properties, it's an object.
		if _, hasProps := sanitized["properties"]; hasProps {
			sanitized["type"] = "object"
		}
	}

	return sanitized
}

// mapClaudeRoleToGemini maps Claude roles to Gemini roles
func (c *GeminiConverter) mapClaudeRoleToGemini(role string) string {
	switch role {
	case "user":
		return "user"
	case "assistant":
		return "model"
	default:
		return "user"
	}
}

// ConvertFromOpenAI converts OpenAI format request to Gemini format
func (c *GeminiConverter) ConvertFromOpenAI(openaiReq *models.OpenAIChatCompletionRequest) (*models.GeminiRequest, error) {
	// This is a placeholder implementation - convert OpenAI to Claude first, then to Gemini
	// For now, return a basic implementation
	geminiReq := &models.GeminiRequest{
		Contents: []models.GeminiContent{},
	}

	// Convert OpenAI messages to Gemini format
	for _, msg := range openaiReq.Messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}

		geminiReq.Contents = append(geminiReq.Contents, models.GeminiContent{
			Role:  role,
			Parts: []models.GeminiPart{{Text: msg.Content}},
		})
	}

	// Handle generation config
	config := &models.GeminiGenerationConfig{}
	if openaiReq.Temperature != nil {
		config.Temperature = openaiReq.Temperature
	}
	if openaiReq.MaxTokens != nil {
		config.MaxOutputTokens = openaiReq.MaxTokens
	}
	geminiReq.GenerationConfig = config

	return geminiReq, nil
}

// ConvertToOpenAI converts Gemini format request to OpenAI format
func (c *GeminiConverter) ConvertToOpenAI(geminiReq *models.GeminiRequest) (*models.OpenAIChatCompletionRequest, error) {
	// This is a placeholder implementation
	openaiReq := &models.OpenAIChatCompletionRequest{
		Messages: []models.Message{},
	}

	// Convert Gemini contents to OpenAI messages
	for _, content := range geminiReq.Contents {
		role := "user"
		if content.Role == "model" {
			role = "assistant"
		}

		// Combine all text parts
		var text string
		for _, part := range content.Parts {
			if part.Text != "" {
				text += part.Text
			}
		}

		if text != "" {
			openaiReq.Messages = append(openaiReq.Messages, models.Message{
				Role:    role,
				Content: text,
			})
		}
	}

	return openaiReq, nil
}

// ConvertToClaude converts Gemini response to Claude format
func (c *GeminiConverter) ConvertToClaude(geminiResp *models.GeminiResponse, originalReq *models.ClaudeMessagesRequest) (*models.ClaudeResponse, error) {
	if geminiResp == nil || len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("empty Gemini response")
	}

	candidate := geminiResp.Candidates[0]
	claudeResp := &models.ClaudeResponse{
		ID:      "msg_" + generateRandomID(),
		Type:    "message",
		Role:    "assistant",
		Content: []models.ClaudeContentBlock{},
		Model:   originalReq.Model,
	}

	// Convert content parts
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			claudeResp.Content = append(claudeResp.Content, models.ClaudeContentBlock{
				Type: "text",
				Text: part.Text,
			})
		}

		if part.FunctionCall != nil {
			claudeResp.Content = append(claudeResp.Content, models.ClaudeContentBlock{
				Type:  "tool_use",
				ID:    generateRandomID(),
				Name:  part.FunctionCall.Name,
				Input: part.FunctionCall.Args,
			})
		}
	}

	// Map finish reason
	switch candidate.FinishReason {
	case "STOP":
		claudeResp.StopReason = "end_turn"
	case "MAX_TOKENS":
		claudeResp.StopReason = "max_tokens"
	default:
		claudeResp.StopReason = "end_turn"
	}

	// Add usage information
	if geminiResp.UsageMetadata != nil {
		claudeResp.Usage = models.ClaudeUsage{
			InputTokens:  geminiResp.UsageMetadata.PromptTokenCount,
			OutputTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
		}
	}

	return claudeResp, nil
}

// generateRandomID generates a random ID for Claude responses
func generateRandomID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
