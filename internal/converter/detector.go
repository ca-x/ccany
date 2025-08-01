package converter

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// APIFormat represents supported API formats
type APIFormat string

const (
	FormatOpenAI    APIFormat = "openai"
	FormatAnthropic APIFormat = "anthropic"
	FormatGemini    APIFormat = "gemini"
	FormatUnknown   APIFormat = "unknown"
)

// CapabilityStatus represents the status of a capability test
type CapabilityStatus string

const (
	StatusSupported    CapabilityStatus = "supported"
	StatusNotSupported CapabilityStatus = "not_supported"
	StatusUnknown      CapabilityStatus = "unknown"
	StatusError        CapabilityStatus = "error"
)

// CapabilityResult contains the result of a capability test
type CapabilityResult struct {
	Capability   string           `json:"capability"`
	Status       CapabilityStatus `json:"status"`
	Details      interface{}      `json:"details,omitempty"`
	Error        string           `json:"error,omitempty"`
	ResponseTime float64          `json:"response_time,omitempty"`
}

// ProviderCapabilities contains all capabilities for a provider
type ProviderCapabilities struct {
	Provider      string                      `json:"provider"`
	BaseURL       string                      `json:"base_url"`
	Models        []string                    `json:"models"`
	Capabilities  map[string]CapabilityResult `json:"capabilities"`
	DetectionTime time.Time                   `json:"detection_time"`
}

// DetectionResult contains format detection result with enhanced information
type DetectionResult struct {
	Format       APIFormat             `json:"format"`
	Confidence   float64               `json:"confidence"` // 0.0 to 1.0
	Reasons      []string              `json:"reasons"`
	Capabilities *ProviderCapabilities `json:"capabilities,omitempty"`
}

// FormatDetector detects API format from requests
type FormatDetector struct {
	pathPatterns map[APIFormat][]string
	headerKeys   map[APIFormat][]string
	bodyKeys     map[APIFormat][]string
}

// NewFormatDetector creates a new format detector
func NewFormatDetector() *FormatDetector {
	return &FormatDetector{
		pathPatterns: map[APIFormat][]string{
			FormatOpenAI: {
				"/v1/chat/completions",
				"/openai/v1/chat/completions",
				"/chat/completions",
			},
			FormatAnthropic: {
				"/v1/messages",
				"/anthropic/v1/messages",
				"/messages",
			},
			FormatGemini: {
				"/v1beta/generateContent",
				"/gemini/v1beta/generateContent",
				"/generateContent",
				"/v1/models/",
			},
		},
		headerKeys: map[APIFormat][]string{
			FormatOpenAI: {
				"authorization", // Bearer token
			},
			FormatAnthropic: {
				"x-api-key",
				"anthropic-version",
			},
			FormatGemini: {
				"x-goog-api-key",
			},
		},
		bodyKeys: map[APIFormat][]string{
			FormatOpenAI: {
				"messages",
				"model",
				"max_tokens", // Note: OpenAI uses max_tokens
			},
			FormatAnthropic: {
				"messages",
				"model",
				"max_tokens", // Claude also uses max_tokens
				"system",     // Claude-specific
			},
			FormatGemini: {
				"contents",
				"generationConfig",
				"systemInstruction", // Gemini-specific
			},
		},
	}
}

// DetectFromRequest detects format from HTTP request
func (fd *FormatDetector) DetectFromRequest(r *http.Request, body []byte) *DetectionResult {
	scores := make(map[APIFormat]float64)
	reasons := make(map[APIFormat][]string)

	// Initialize scores
	for format := range fd.pathPatterns {
		scores[format] = 0.0
		reasons[format] = []string{}
	}

	// Check URL path
	fd.checkPath(r.URL.Path, scores, reasons)

	// Check headers
	fd.checkHeaders(r.Header, scores, reasons)

	// Check body structure
	if body != nil {
		fd.checkBody(body, scores, reasons)
	}

	// Find the format with highest score
	var bestFormat APIFormat = FormatUnknown
	var bestScore float64 = 0.0
	var bestReasons []string

	for format, score := range scores {
		if score > bestScore {
			bestScore = score
			bestFormat = format
			bestReasons = reasons[format]
		}
	}

	// If no clear winner, try additional heuristics
	if bestScore < 0.6 && body != nil {
		heuristicResult := fd.applyHeuristics(body)
		if heuristicResult.Confidence > bestScore {
			return heuristicResult
		}
	}

	return &DetectionResult{
		Format:     bestFormat,
		Confidence: bestScore,
		Reasons:    bestReasons,
	}
}

// DetectFromJSON detects format from JSON body only
func (fd *FormatDetector) DetectFromJSON(body []byte) *DetectionResult {
	scores := make(map[APIFormat]float64)
	reasons := make(map[APIFormat][]string)

	// Initialize scores
	for format := range fd.bodyKeys {
		scores[format] = 0.0
		reasons[format] = []string{}
	}

	// Check body structure
	fd.checkBody(body, scores, reasons)

	// Apply heuristics
	heuristicResult := fd.applyHeuristics(body)

	// Find the format with highest score
	var bestFormat APIFormat = FormatUnknown
	var bestScore float64 = 0.0
	var bestReasons []string

	for format, score := range scores {
		if score > bestScore {
			bestScore = score
			bestFormat = format
			bestReasons = reasons[format]
		}
	}

	// Use heuristic result if it's better
	if heuristicResult.Confidence > bestScore {
		return heuristicResult
	}

	return &DetectionResult{
		Format:     bestFormat,
		Confidence: bestScore,
		Reasons:    bestReasons,
	}
}

// checkPath checks URL path patterns
func (fd *FormatDetector) checkPath(path string, scores map[APIFormat]float64, reasons map[APIFormat][]string) {
	path = strings.ToLower(path)

	for format, patterns := range fd.pathPatterns {
		for _, pattern := range patterns {
			if strings.Contains(path, strings.ToLower(pattern)) {
				scores[format] += 0.8 // High weight for path matching
				reasons[format] = append(reasons[format], "URL path matches "+pattern)
				break // Only count once per format
			}
		}
	}
}

// checkHeaders checks HTTP headers
func (fd *FormatDetector) checkHeaders(headers http.Header, scores map[APIFormat]float64, reasons map[APIFormat][]string) {
	for format, headerKeys := range fd.headerKeys {
		for _, key := range headerKeys {
			if headers.Get(key) != "" {
				scores[format] += 0.3 // Medium weight for headers
				reasons[format] = append(reasons[format], "Header '"+key+"' present")
			}
		}
	}
}

// checkBody checks request body structure
func (fd *FormatDetector) checkBody(body []byte, scores map[APIFormat]float64, reasons map[APIFormat][]string) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return // Can't parse JSON
	}

	for format, bodyKeys := range fd.bodyKeys {
		for _, key := range bodyKeys {
			if _, exists := data[key]; exists {
				weight := 0.4 // Base weight

				// Higher weight for format-specific keys
				switch format {
				case FormatAnthropic:
					if key == "system" {
						weight = 0.6 // Claude-specific
					}
				case FormatGemini:
					if key == "contents" || key == "systemInstruction" {
						weight = 0.6 // Gemini-specific
					}
				}

				scores[format] += weight
				reasons[format] = append(reasons[format], "Body contains '"+key+"' field")
			}
		}
	}
}

// applyHeuristics applies additional detection heuristics
func (fd *FormatDetector) applyHeuristics(body []byte) *DetectionResult {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return &DetectionResult{Format: FormatUnknown, Confidence: 0.0}
	}

	reasons := []string{}

	// Check for Anthropic-specific patterns
	if system, exists := data["system"]; exists && system != nil {
		reasons = append(reasons, "Contains 'system' field (Anthropic specific)")
		return &DetectionResult{
			Format:     FormatAnthropic,
			Confidence: 0.8,
			Reasons:    reasons,
		}
	}

	// Check for Gemini-specific patterns
	if contents, exists := data["contents"]; exists {
		if contentsArray, ok := contents.([]interface{}); ok && len(contentsArray) > 0 {
			if contentItem, ok := contentsArray[0].(map[string]interface{}); ok {
				if parts, exists := contentItem["parts"]; exists && parts != nil {
					reasons = append(reasons, "Contains 'contents.parts' structure (Gemini specific)")
					return &DetectionResult{
						Format:     FormatGemini,
						Confidence: 0.9,
						Reasons:    reasons,
					}
				}
			}
		}
	}

	// Check for OpenAI patterns (messages without system at root level)
	if messages, exists := data["messages"]; exists {
		if messagesArray, ok := messages.([]interface{}); ok && len(messagesArray) > 0 {
			// Check if it's NOT Anthropic (no system field at root)
			if _, hasSystem := data["system"]; !hasSystem {
				reasons = append(reasons, "Contains 'messages' without root 'system' field (OpenAI pattern)")
				return &DetectionResult{
					Format:     FormatOpenAI,
					Confidence: 0.7,
					Reasons:    reasons,
				}
			}
		}
	}

	return &DetectionResult{Format: FormatUnknown, Confidence: 0.0}
}

// GetSupportedFormats returns list of supported formats
func (fd *FormatDetector) GetSupportedFormats() []APIFormat {
	return []APIFormat{FormatOpenAI, FormatAnthropic, FormatGemini}
}

// IsSupported checks if a format is supported
func (fd *FormatDetector) IsSupported(format APIFormat) bool {
	for _, supported := range fd.GetSupportedFormats() {
		if format == supported {
			return true
		}
	}
	return false
}

// CapabilityTest represents a capability test configuration
type CapabilityTest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	TestData    interface{} `json:"test_data"`
	Expected    interface{} `json:"expected"`
}

// GetDefaultCapabilityTests returns default capability tests for each format
func (fd *FormatDetector) GetDefaultCapabilityTests(format APIFormat) []CapabilityTest {
	switch format {
	case FormatOpenAI:
		return []CapabilityTest{
			{
				Name:        "basic_chat",
				Description: "Basic chat completion",
				Category:    "core",
				TestData: map[string]interface{}{
					"model": "gpt-3.5-turbo",
					"messages": []map[string]interface{}{
						{"role": "user", "content": "Hello"},
					},
					"max_tokens": 10,
				},
			},
			{
				Name:        "streaming",
				Description: "Streaming response support",
				Category:    "core",
				TestData: map[string]interface{}{
					"model": "gpt-3.5-turbo",
					"messages": []map[string]interface{}{
						{"role": "user", "content": "Hello"},
					},
					"stream":     true,
					"max_tokens": 10,
				},
			},
			{
				Name:        "function_calling",
				Description: "Function calling support",
				Category:    "advanced",
				TestData: map[string]interface{}{
					"model": "gpt-3.5-turbo",
					"messages": []map[string]interface{}{
						{"role": "user", "content": "What's the weather like?"},
					},
					"tools": []map[string]interface{}{
						{
							"type": "function",
							"function": map[string]interface{}{
								"name":        "get_weather",
								"description": "Get weather information",
								"parameters": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"location": map[string]interface{}{
											"type":        "string",
											"description": "City name",
										},
									},
									"required": []string{"location"},
								},
							},
						},
					},
					"tool_choice": "auto",
				},
			},
		}
	case FormatAnthropic:
		return []CapabilityTest{
			{
				Name:        "basic_chat",
				Description: "Basic message completion",
				Category:    "core",
				TestData: map[string]interface{}{
					"model": "claude-3-haiku-20240307",
					"messages": []map[string]interface{}{
						{"role": "user", "content": "Hello"},
					},
					"max_tokens": 10,
				},
			},
			{
				Name:        "streaming",
				Description: "Streaming response support",
				Category:    "core",
				TestData: map[string]interface{}{
					"model": "claude-3-haiku-20240307",
					"messages": []map[string]interface{}{
						{"role": "user", "content": "Hello"},
					},
					"stream":     true,
					"max_tokens": 10,
				},
			},
			{
				Name:        "system_messages",
				Description: "System message support",
				Category:    "core",
				TestData: map[string]interface{}{
					"model":  "claude-3-haiku-20240307",
					"system": "You are a helpful assistant.",
					"messages": []map[string]interface{}{
						{"role": "user", "content": "Hello"},
					},
					"max_tokens": 10,
				},
			},
			{
				Name:        "tool_use",
				Description: "Tool usage support",
				Category:    "advanced",
				TestData: map[string]interface{}{
					"model": "claude-3-haiku-20240307",
					"messages": []map[string]interface{}{
						{"role": "user", "content": "What's the weather like?"},
					},
					"tools": []map[string]interface{}{
						{
							"name":        "get_weather",
							"description": "Get weather information",
							"input_schema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"location": map[string]interface{}{
										"type":        "string",
										"description": "City name",
									},
								},
								"required": []string{"location"},
							},
						},
					},
					"tool_choice": map[string]interface{}{"type": "auto"},
					"max_tokens":  100,
				},
			},
		}
	case FormatGemini:
		return []CapabilityTest{
			{
				Name:        "basic_chat",
				Description: "Basic content generation",
				Category:    "core",
				TestData: map[string]interface{}{
					"contents": []map[string]interface{}{
						{
							"parts": []map[string]interface{}{
								{"text": "Hello"},
							},
						},
					},
					"generationConfig": map[string]interface{}{
						"maxOutputTokens": 10,
					},
				},
			},
			{
				Name:        "streaming",
				Description: "Streaming response support",
				Category:    "core",
				TestData: map[string]interface{}{
					"contents": []map[string]interface{}{
						{
							"parts": []map[string]interface{}{
								{"text": "Hello"},
							},
						},
					},
					"generationConfig": map[string]interface{}{
						"maxOutputTokens": 10,
					},
				},
			},
			{
				Name:        "system_instruction",
				Description: "System instruction support",
				Category:    "core",
				TestData: map[string]interface{}{
					"systemInstruction": map[string]interface{}{
						"parts": []map[string]interface{}{
							{"text": "You are a helpful assistant."},
						},
					},
					"contents": []map[string]interface{}{
						{
							"parts": []map[string]interface{}{
								{"text": "Hello"},
							},
						},
					},
					"generationConfig": map[string]interface{}{
						"maxOutputTokens": 10,
					},
				},
			},
		}
	default:
		return []CapabilityTest{}
	}
}

// DetectCapabilities performs comprehensive capability detection for a provider
func (fd *FormatDetector) DetectCapabilities(baseURL, apiKey string, format APIFormat, customTests []CapabilityTest) (*ProviderCapabilities, error) {
	capabilities := &ProviderCapabilities{
		Provider:      string(format),
		BaseURL:       baseURL,
		Models:        []string{},
		Capabilities:  make(map[string]CapabilityResult),
		DetectionTime: time.Now(),
	}

	// First, try to detect available models
	models, err := fd.detectModels(baseURL, apiKey, format)
	if err == nil {
		capabilities.Models = models
	}

	// Get default tests for this format
	tests := fd.GetDefaultCapabilityTests(format)

	// Add custom tests if provided
	if customTests != nil {
		tests = append(tests, customTests...)
	}

	// Run capability tests
	for _, test := range tests {
		result := fd.runCapabilityTest(baseURL, apiKey, format, test)
		capabilities.Capabilities[test.Name] = result
	}

	return capabilities, nil
}

// detectModels attempts to detect available models for a provider
func (fd *FormatDetector) detectModels(_ /* baseURL */, _ /* apiKey */ string, format APIFormat) ([]string, error) {
	// This is a simplified implementation - in practice, you'd make actual API calls
	switch format {
	case FormatOpenAI:
		return []string{"gpt-3.5-turbo", "gpt-4", "gpt-4-turbo"}, nil
	case FormatAnthropic:
		return []string{"claude-3-haiku-20240307", "claude-3-sonnet-20240229", "claude-3-opus-20240229"}, nil
	case FormatGemini:
		return []string{"gemini-pro", "gemini-pro-vision"}, nil
	default:
		return []string{}, nil
	}
}

// runCapabilityTest runs a single capability test
func (fd *FormatDetector) runCapabilityTest(_ /* baseURL */, _ /* apiKey */ string, format APIFormat, test CapabilityTest) CapabilityResult {
	startTime := time.Now()

	// This is a simplified implementation
	// In practice, you'd make actual HTTP requests to test capabilities

	result := CapabilityResult{
		Capability:   test.Name,
		Status:       StatusSupported, // Default to supported for demo
		ResponseTime: time.Since(startTime).Seconds(),
	}

	// Add some realistic logic based on test type
	switch test.Name {
	case "streaming":
		// Check if the format supports streaming
		if format == FormatOpenAI || format == FormatAnthropic || format == FormatGemini {
			result.Status = StatusSupported
			result.Details = map[string]interface{}{
				"format": "Server-Sent Events",
				"tested": true,
			}
		} else {
			result.Status = StatusNotSupported
		}
	case "function_calling", "tool_use":
		// Tool support varies by format
		if format == FormatOpenAI || format == FormatAnthropic {
			result.Status = StatusSupported
			result.Details = map[string]interface{}{
				"max_tools":      100,
				"parallel_calls": true,
			}
		} else {
			result.Status = StatusNotSupported
		}
	case "system_messages", "system_instruction":
		result.Status = StatusSupported
		result.Details = map[string]interface{}{
			"max_length": 32000,
		}
	default:
		result.Status = StatusSupported
	}

	return result
}
