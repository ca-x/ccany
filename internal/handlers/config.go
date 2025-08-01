package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ccany/internal/app"
	"ccany/internal/client"
	"ccany/internal/config"
	"ccany/internal/i18n"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ConfigHandler configuration management handler
type ConfigHandler struct {
	configManager *app.ConfigManager
	logger        *logrus.Logger
}

// NewConfigHandler creates configuration management handler
func NewConfigHandler(configManager *app.ConfigManager, logger *logrus.Logger) *ConfigHandler {
	return &ConfigHandler{
		configManager: configManager,
		logger:        logger,
	}
}

// ConfigItem configuration item response structure
type ConfigItem struct {
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	Category    string      `json:"category"`
	Type        string      `json:"type"`
	IsEncrypted bool        `json:"is_encrypted"`
	IsRequired  bool        `json:"is_required"`
	Description string      `json:"description"`
}

// ConfigUpdateRequest configuration update request
type ConfigUpdateRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}

// BulkConfigUpdateRequest bulk configuration update request
type BulkConfigUpdateRequest struct {
	OpenAIAPIKey     string `json:"openai_api_key"`
	ClaudeAPIKey     string `json:"claude_api_key"`
	OpenAIBaseURL    string `json:"openai_base_url"`
	ClaudeBaseURL    string `json:"claude_base_url"`
	BigModel         string `json:"big_model"`
	SmallModel       string `json:"small_model"`
	MaxTokensLimit   int    `json:"max_tokens_limit"`
	RequestTimeout   int    `json:"request_timeout"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	LogLevel         string `json:"log_level"`
	JWTSecret        string `json:"jwt_secret"`
	EncryptAlgorithm string `json:"encrypt_algorithm"`
	// 代理配置字段
	ProxyEnabled          bool   `json:"proxy_enabled"`
	ProxyType             string `json:"proxy_type"`
	HTTPProxy             string `json:"http_proxy"`
	SOCKS5Proxy           string `json:"socks5_proxy"`
	SOCKS5ProxyUser       string `json:"socks5_proxy_user"`
	SOCKS5ProxyPassword   string `json:"socks5_proxy_password"`
	IgnoreSSLVerification bool   `json:"ignore_ssl_verification"`
}

// GetAllConfigs gets all configurations - GET /admin/configs
func (h *ConfigHandler) GetAllConfigs(c *gin.Context) {
	h.logger.Info("Getting all configs")
	configs, err := h.configManager.GetAllConfigs()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get all configs")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": i18n.T(c, "errors.config_error"),
		})
		return
	}

	h.logger.WithField("config_count", len(configs)).Info("Retrieved configs")

	// convert to response format
	var configItems []ConfigItem
	for key, valueData := range configs {
		var displayValue interface{}

		// handle complex structure returned from service
		if valueMap, ok := valueData.(map[string]interface{}); ok {
			if isEncrypted, exists := valueMap["is_encrypted"]; exists {
				if encrypted, ok := isEncrypted.(bool); ok && encrypted {
					// encrypted configuration, check if already set
					if isSet, exists := valueMap["is_set"]; exists {
						if set, ok := isSet.(bool); ok && set {
							displayValue = "***masked***"
						} else {
							displayValue = ""
						}
					} else {
						displayValue = ""
					}
				} else {
					// non-encrypted configuration, get value directly
					if value, exists := valueMap["value"]; exists {
						displayValue = value
					} else {
						displayValue = ""
					}
				}
			} else {
				// non-encrypted configuration, get value directly
				if value, exists := valueMap["value"]; exists {
					displayValue = value
				} else {
					displayValue = ""
				}
			}
		} else {
			// if not expected structure, use original value directly
			displayValue = valueData
		}

		configItems = append(configItems, ConfigItem{
			Key:         key,
			Value:       displayValue,
			Category:    h.getConfigCategory(key),
			Type:        h.getConfigType(key),
			IsEncrypted: h.isEncryptedConfig(key),
			IsRequired:  h.isRequiredConfig(key),
			Description: h.getConfigDescription(key),
		})
	}

	h.logger.WithField("config_items_count", len(configItems)).Info("Converted config items")

	c.JSON(http.StatusOK, gin.H{
		"configs": configItems,
	})
}

// GetConfigObject gets configuration as a single object - GET /admin/config
func (h *ConfigHandler) GetConfigObject(c *gin.Context) {
	h.logger.Info("Getting config object")

	// Get configuration using config manager
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get config object")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": i18n.T(c, "errors.config_error"),
		})
		return
	}

	// Convert config struct to response format expected by frontend
	configResponse := gin.H{
		"openai_api_key":   cfg.OpenAIAPIKey,
		"claude_api_key":   cfg.ClaudeAPIKey,
		"openai_base_url":  cfg.OpenAIBaseURL,
		"claude_base_url":  cfg.ClaudeBaseURL,
		"big_model":        cfg.BigModel,
		"small_model":      cfg.SmallModel,
		"max_tokens_limit": cfg.MaxTokensLimit,
		"request_timeout":  cfg.RequestTimeout,
		"host":             cfg.Host,
		"port":             cfg.Port,
		"log_level":        cfg.LogLevel,
		"temperature":      cfg.Temperature,
		"stream_enabled":   cfg.StreamEnabled,
	}

	// Get additional configuration fields that might not be in the main Config struct
	jwtSecret, _ := h.configManager.GetConfigValue("jwt_secret")
	encryptAlgo, _ := h.configManager.GetConfigValue("encrypt_algorithm")

	// Get proxy configuration fields
	proxyEnabled, _ := h.configManager.GetConfigValue(config.KeyProxyEnabled)
	proxyType, _ := h.configManager.GetConfigValue(config.KeyProxyType)
	httpProxy, _ := h.configManager.GetConfigValue(config.KeyHTTPProxy)
	socks5Proxy, _ := h.configManager.GetConfigValue(config.KeySOCKS5Proxy)
	socks5ProxyUser, _ := h.configManager.GetConfigValue(config.KeySOCKS5ProxyUser)
	socks5ProxyPassword, _ := h.configManager.GetConfigValue(config.KeySOCKS5ProxyPassword)
	ignoreSSL, _ := h.configManager.GetConfigValue(config.KeyIgnoreSSLVerification)

	// Add these fields to response
	configResponse["jwt_secret"] = jwtSecret
	configResponse["encrypt_algorithm"] = encryptAlgo
	configResponse["proxy_enabled"] = proxyEnabled
	configResponse["proxy_type"] = proxyType
	configResponse["http_proxy"] = httpProxy
	configResponse["socks5_proxy"] = socks5Proxy
	configResponse["socks5_proxy_user"] = socks5ProxyUser
	configResponse["socks5_proxy_password"] = socks5ProxyPassword
	configResponse["ignore_ssl_verification"] = ignoreSSL

	h.logger.Info("Returning config object")
	c.JSON(http.StatusOK, gin.H{
		"config": configResponse,
	})
}

// GetConfig gets single configuration - GET /admin/configs/:key
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": i18n.T(c, "errors.missing_parameter"),
		})
		return
	}

	value, err := h.configManager.GetConfigValue(key)
	if err != nil {
		h.logger.WithError(err).WithField("key", key).Error("Failed to get config")
		c.JSON(http.StatusNotFound, gin.H{
			"error": i18n.T(c, "errors.config_not_found"),
		})
		return
	}

	configItem := ConfigItem{
		Key:         key,
		Value:       h.maskSensitiveValue(key, value),
		Category:    h.getConfigCategory(key),
		Type:        h.getConfigType(key),
		IsEncrypted: h.isEncryptedConfig(key),
		IsRequired:  h.isRequiredConfig(key),
		Description: h.getConfigDescription(key),
	}

	c.JSON(http.StatusOK, gin.H{
		"config": configItem,
	})
}

// UpdateConfig updates configuration - PUT /admin/configs/:key
func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Configuration key is required",
		})
		return
	}

	var req ConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// verify configuration key consistency
	if req.Key != key {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Configuration key mismatch",
		})
		return
	}

	// validate configuration value
	if err := h.validateConfigValue(key, req.Value); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// update configuration
	encrypted := h.isEncryptedConfig(key)
	if err := h.configManager.UpdateConfig(key, req.Value, encrypted); err != nil {
		h.logger.WithError(err).WithField("key", key).Error("Failed to update config")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update configuration",
		})
		return
	}

	h.logger.WithField("key", key).Info("Configuration updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration updated successfully",
		"key":     key,
	})
}

// UpdateBulkConfig updates multiple configurations - PUT /admin/config
func (h *ConfigHandler) UpdateBulkConfig(c *gin.Context) {
	var req BulkConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid bulk config request format")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"openai_api_key":   req.OpenAIAPIKey != "",
		"claude_api_key":   req.ClaudeAPIKey != "",
		"openai_base_url":  req.OpenAIBaseURL,
		"big_model":        req.BigModel,
		"small_model":      req.SmallModel,
		"max_tokens_limit": req.MaxTokensLimit,
		"request_timeout":  req.RequestTimeout,
		"host":             req.Host,
		"port":             req.Port,
		"log_level":        req.LogLevel,
	}).Info("Updating bulk configuration with received data")

	// Map of field names to their config keys and whether they're encrypted
	configUpdates := map[string]struct {
		key          string
		value        string
		encrypted    bool
		shouldUpdate bool
	}{
		"openai_api_key":    {key: config.KeyOpenAIAPIKey, value: req.OpenAIAPIKey, encrypted: false, shouldUpdate: req.OpenAIAPIKey != ""},
		"claude_api_key":    {key: config.KeyClaudeAPIKey, value: req.ClaudeAPIKey, encrypted: false, shouldUpdate: req.ClaudeAPIKey != ""},
		"openai_base_url":   {key: config.KeyOpenAIBaseURL, value: req.OpenAIBaseURL, encrypted: false, shouldUpdate: req.OpenAIBaseURL != ""},
		"claude_base_url":   {key: config.KeyClaudeBaseURL, value: req.ClaudeBaseURL, encrypted: false, shouldUpdate: req.ClaudeBaseURL != ""},
		"big_model":         {key: config.KeyBigModel, value: req.BigModel, encrypted: false, shouldUpdate: req.BigModel != ""},
		"small_model":       {key: config.KeySmallModel, value: req.SmallModel, encrypted: false, shouldUpdate: req.SmallModel != ""},
		"max_tokens_limit":  {key: config.KeyMaxTokens, value: strconv.Itoa(req.MaxTokensLimit), encrypted: false, shouldUpdate: req.MaxTokensLimit > 0},
		"request_timeout":   {key: config.KeyRequestTimeout, value: strconv.Itoa(req.RequestTimeout), encrypted: false, shouldUpdate: req.RequestTimeout > 0},
		"host":              {key: config.KeyServerHost, value: req.Host, encrypted: false, shouldUpdate: req.Host != ""},
		"port":              {key: config.KeyServerPort, value: strconv.Itoa(req.Port), encrypted: false, shouldUpdate: req.Port > 0},
		"log_level":         {key: config.KeyLogLevel, value: req.LogLevel, encrypted: false, shouldUpdate: req.LogLevel != ""},
		"jwt_secret":        {key: "jwt_secret", value: req.JWTSecret, encrypted: false, shouldUpdate: req.JWTSecret != ""},
		"encrypt_algorithm": {key: "encrypt_algorithm", value: req.EncryptAlgorithm, encrypted: false, shouldUpdate: req.EncryptAlgorithm != ""},
		// 代理配置
		"proxy_enabled":           {key: config.KeyProxyEnabled, value: strconv.FormatBool(req.ProxyEnabled), encrypted: false, shouldUpdate: true},
		"proxy_type":              {key: config.KeyProxyType, value: req.ProxyType, encrypted: false, shouldUpdate: req.ProxyType != ""},
		"http_proxy":              {key: config.KeyHTTPProxy, value: req.HTTPProxy, encrypted: false, shouldUpdate: req.HTTPProxy != ""},
		"socks5_proxy":            {key: config.KeySOCKS5Proxy, value: req.SOCKS5Proxy, encrypted: false, shouldUpdate: req.SOCKS5Proxy != ""},
		"socks5_proxy_user":       {key: config.KeySOCKS5ProxyUser, value: req.SOCKS5ProxyUser, encrypted: false, shouldUpdate: req.SOCKS5ProxyUser != ""},
		"socks5_proxy_password":   {key: config.KeySOCKS5ProxyPassword, value: req.SOCKS5ProxyPassword, encrypted: true, shouldUpdate: req.SOCKS5ProxyPassword != ""},
		"ignore_ssl_verification": {key: config.KeyIgnoreSSLVerification, value: strconv.FormatBool(req.IgnoreSSLVerification), encrypted: false, shouldUpdate: true},
	}

	// Validate all values first
	for fieldName, update := range configUpdates {
		if update.shouldUpdate && update.value != "" {
			if err := h.validateConfigValue(update.key, update.value); err != nil {
				h.logger.WithError(err).WithField("field", fieldName).Error("Validation failed for config field")
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Validation failed for %s: %s", fieldName, err.Error()),
				})
				return
			}
		}
	}

	// Update all configurations
	updatedCount := 0
	var updateErrors []string

	for fieldName, update := range configUpdates {
		if update.shouldUpdate {
			h.logger.WithFields(logrus.Fields{
				"field": fieldName,
				"key":   update.key,
				"value": func() string {
					if update.encrypted && update.value != "" {
						return "***masked***"
					}
					return update.value
				}(),
				"encrypted": update.encrypted,
			}).Debug("Updating config field")

			if err := h.configManager.UpdateConfig(update.key, update.value, update.encrypted); err != nil {
				h.logger.WithError(err).WithField("field", fieldName).Error("Failed to update config field")
				updateErrors = append(updateErrors, fmt.Sprintf("%s: %s", fieldName, err.Error()))
			} else {
				updatedCount++
				h.logger.WithField("key", update.key).Debug("Configuration field updated successfully")
			}
		} else {
			h.logger.WithField("field", fieldName).Debug("Skipping field update (empty or invalid value)")
		}
	}

	if len(updateErrors) > 0 {
		h.logger.WithField("errors", updateErrors).Error("Some configuration updates failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Some configuration updates failed",
			"details": updateErrors,
			"updated": updatedCount,
			"failed":  len(updateErrors),
		})
		return
	}

	h.logger.WithField("updated_count", updatedCount).Info("Bulk configuration update completed successfully")

	// Get updated configuration for response
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		h.logger.WithError(err).Warn("Failed to get updated config for response")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration updated successfully",
		"updated": updatedCount,
		"config":  cfg,
	})
}

// TestConfig tests configuration - POST /admin/configs/test
func (h *ConfigHandler) TestConfig(c *gin.Context) {
	// validate current configuration
	if err := h.configManager.ValidateConfig(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":  false,
			"error":  err.Error(),
			"status": "Configuration validation failed",
		})
		return
	}

	// check if configured
	configured, err := h.configManager.IsConfigured()
	if err != nil {
		h.logger.WithError(err).Error("Failed to check configuration status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check configuration status",
		})
		return
	}

	// 构建与前端兼容的响应格式
	results := []gin.H{
		{
			"service": "Configuration",
			"status":  "Valid",
		},
		{
			"service": "Database",
			"status":  "Connected",
		},
	}

	// 如果配置了API密钥，添加API状态
	cfg, err := h.configManager.GetConfig()
	if err == nil {
		if cfg.OpenAIAPIKey != "" {
			results = append(results, gin.H{
				"service": "OpenAI API",
				"status":  "Configured",
			})
		}
		if cfg.ClaudeAPIKey != "" {
			results = append(results, gin.H{
				"service": "Claude API",
				"status":  "Configured",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":      true,
		"configured": configured,
		"status":     "Configuration is valid",
		"results":    results, // 添加前端期望的results字段
	})
}

// TestAPIKey tests API key - POST /admin/config/test-api-key
func (h *ConfigHandler) TestAPIKey(c *gin.Context) {
	h.logger.Info("Testing API key")

	// get current configuration
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get current config")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get current configuration",
		})
		return
	}

	// check if any API key is configured
	if cfg.OpenAIAPIKey == "" && cfg.ClaudeAPIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No API keys configured. Please configure at least one API key (OpenAI or Claude)",
		})
		return
	}

	// test API key
	results := h.testAPIConnectivity(cfg)

	// check if any test was successful
	hasSuccess := false
	for _, result := range results {
		if result.Success {
			hasSuccess = true
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": hasSuccess,
		"results": results,
		"message": func() string {
			if hasSuccess {
				return "API key test successful"
			}
			return "API key test failed"
		}(),
	})
}

// APITestResult API test result structure
type APITestResult struct {
	Service      string `json:"service"`
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	Model        string `json:"model,omitempty"`
	ResponseTime string `json:"response_time,omitempty"`
	Error        string `json:"error,omitempty"`
}

// testAPIConnectivity tests API connectivity
func (h *ConfigHandler) testAPIConnectivity(cfg *config.Config) []APITestResult {
	var results []APITestResult

	// test OpenAI API
	if cfg.OpenAIAPIKey != "" {
		result := h.testOpenAIAPI(cfg)
		results = append(results, result)
	}

	// also test Claude API if configured
	if cfg.ClaudeAPIKey != "" {
		result := h.testClaudeAPI(cfg)
		results = append(results, result)
	}

	return results
}

// testOpenAIAPI tests OpenAI API
func (h *ConfigHandler) testOpenAIAPI(cfg *config.Config) APITestResult {
	startTime := time.Now()

	// create simple test request
	testPayload := map[string]interface{}{
		"model":      cfg.BigModel,
		"messages":   []map[string]string{{"role": "user", "content": "Hello"}},
		"max_tokens": 1,
	}

	// create HTTP client
	client := &http.Client{
		Timeout: time.Duration(cfg.RequestTimeout) * time.Second,
	}

	// create request
	jsonData, _ := json.Marshal(testPayload)
	req, err := http.NewRequest("POST", cfg.OpenAIBaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return APITestResult{
			Service: "OpenAI API",
			Success: false,
			Error:   fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	// set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIAPIKey)

	// send request
	resp, err := client.Do(req)
	if err != nil {
		return APITestResult{
			Service: "OpenAI API",
			Success: false,
			Error:   fmt.Sprintf("Request failed: %v", err),
		}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			h.logger.WithError(err).Error("Failed to close response body")
		}
	}()

	responseTime := time.Since(startTime)

	// check response status
	if resp.StatusCode == http.StatusOK {
		return APITestResult{
			Service:      "OpenAI API",
			Success:      true,
			Message:      "API key valid and responsive",
			Model:        cfg.BigModel,
			ResponseTime: fmt.Sprintf("%.2fms", float64(responseTime.Nanoseconds())/1000000),
		}
	} else {
		// read error response
		body, _ := io.ReadAll(resp.Body)
		return APITestResult{
			Service: "OpenAI API",
			Success: false,
			Error:   fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		}
	}
}

// testClaudeAPI tests Claude API
func (h *ConfigHandler) testClaudeAPI(cfg *config.Config) APITestResult {
	startTime := time.Now()

	// create simple test request
	testPayload := map[string]interface{}{
		"model":      cfg.BigModel,
		"max_tokens": 1,
		"messages":   []map[string]string{{"role": "user", "content": "Hello"}},
	}

	// create HTTP client
	client := &http.Client{
		Timeout: time.Duration(cfg.RequestTimeout) * time.Second,
	}

	// create request
	jsonData, _ := json.Marshal(testPayload)
	req, err := http.NewRequest("POST", cfg.ClaudeBaseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return APITestResult{
			Service: "Claude API",
			Success: false,
			Error:   fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	// set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.ClaudeAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// send request
	resp, err := client.Do(req)
	if err != nil {
		return APITestResult{
			Service: "Claude API",
			Success: false,
			Error:   fmt.Sprintf("Request failed: %v", err),
		}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			h.logger.WithError(err).Error("Failed to close response body")
		}
	}()

	responseTime := time.Since(startTime)

	// check response status
	if resp.StatusCode == http.StatusOK {
		return APITestResult{
			Service:      "Claude API",
			Success:      true,
			Message:      "API key valid and responsive",
			Model:        cfg.BigModel,
			ResponseTime: fmt.Sprintf("%.2fms", float64(responseTime.Nanoseconds())/1000000),
		}
	} else {
		// read error response
		body, _ := io.ReadAll(resp.Body)
		return APITestResult{
			Service: "Claude API",
			Success: false,
			Error:   fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		}
	}
}

// GetConfigStats gets configuration statistics - GET /admin/configs/stats
func (h *ConfigHandler) GetConfigStats(c *gin.Context) {
	configs, err := h.configManager.GetAllConfigs()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get config stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get configuration statistics",
		})
		return
	}

	var stats = struct {
		Total      int `json:"total"`
		Configured int `json:"configured"`
		Encrypted  int `json:"encrypted"`
		Required   int `json:"required"`
	}{}

	for key, value := range configs {
		stats.Total++
		if value != nil && value != "" {
			stats.Configured++
		}
		if h.isEncryptedConfig(key) {
			stats.Encrypted++
		}
		if h.isRequiredConfig(key) {
			stats.Required++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// maskSensitiveValue masks sensitive values
func (h *ConfigHandler) maskSensitiveValue(key string, value interface{}) interface{} {
	if h.isEncryptedConfig(key) {
		if strValue, ok := value.(string); ok && strValue != "" {
			return "***masked***"
		}
	}
	return value
}

// getConfigCategory gets configuration category
func (h *ConfigHandler) getConfigCategory(key string) string {
	switch key {
	case config.KeyOpenAIAPIKey, config.KeyOpenAIBaseURL, config.KeyClaudeAPIKey, config.KeyClaudeBaseURL:
		return "api"
	case config.KeyServerPort, config.KeyLogLevel:
		return "server"
	case config.KeyMaxTokens, config.KeyTemperature, config.KeyStreamEnabled:
		return "model"
	default:
		return "other"
	}
}

// getConfigType gets configuration type
func (h *ConfigHandler) getConfigType(key string) string {
	switch key {
	case config.KeyServerPort, config.KeyMaxTokens:
		return "int"
	case config.KeyTemperature:
		return "float"
	case config.KeyStreamEnabled:
		return "bool"
	default:
		return "string"
	}
}

// isEncryptedConfig checks if configuration is encrypted
func (h *ConfigHandler) isEncryptedConfig(key string) bool {
	switch key {
	case config.KeyOpenAIAPIKey, config.KeyClaudeAPIKey, "jwt_secret":
		return true
	default:
		return false
	}
}

// isRequiredConfig checks if configuration is required
func (h *ConfigHandler) isRequiredConfig(key string) bool {
	switch key {
	case config.KeyOpenAIAPIKey, config.KeyClaudeAPIKey:
		return true
	default:
		return false
	}
}

// getConfigDescription gets configuration description
func (h *ConfigHandler) getConfigDescription(key string) string {
	descriptions := map[string]string{
		config.KeyOpenAIAPIKey:  "OpenAI API key for accessing OpenAI services",
		config.KeyOpenAIBaseURL: "OpenAI API base URL",
		config.KeyClaudeAPIKey:  "Claude API key for accessing Anthropic services",
		config.KeyClaudeBaseURL: "Claude API base URL",
		config.KeyServerPort:    "Server listening port",
		config.KeyLogLevel:      "Log level (DEBUG/INFO/WARN/ERROR)",
		config.KeyMaxTokens:     "Maximum token limit",
		config.KeyTemperature:   "Model temperature parameter (0.0-1.0)",
		config.KeyStreamEnabled: "Whether to enable streaming response",
	}

	if desc, exists := descriptions[key]; exists {
		return desc
	}
	return "Configuration item"
}

// validateConfigValue validates configuration value
func (h *ConfigHandler) validateConfigValue(key, value string) error {
	switch key {
	case config.KeyServerPort:
		if port, err := strconv.Atoi(value); err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid port number: must be between 1 and 65535")
		}
	case config.KeyMaxTokens:
		if tokens, err := strconv.Atoi(value); err != nil || tokens < 1 {
			return fmt.Errorf("invalid token limit: must be a positive integer")
		}
	case config.KeyTemperature:
		if temp, err := strconv.ParseFloat(value, 64); err != nil || temp < 0 || temp > 1 {
			return fmt.Errorf("temperature must be between 0 and 1")
		}
	case config.KeyStreamEnabled:
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("invalid boolean value: must be true or false")
		}
	}
	return nil
}

// GetFinalEndpointURL returns the final constructed endpoint URL - GET /admin/config/endpoint-url
func (h *ConfigHandler) GetFinalEndpointURL(c *gin.Context) {
	baseURL := c.Query("base_url")
	if baseURL == "" {
		// Get current config base URL if not provided
		cfg, err := h.configManager.GetConfig()
		if err != nil {
			h.logger.WithError(err).Error("Failed to get current config")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get current configuration",
			})
			return
		}
		baseURL = cfg.OpenAIBaseURL
	}

	// Use the same URL construction logic as the client
	finalURL := constructFinalEndpointURL(baseURL)

	c.JSON(http.StatusOK, gin.H{
		"base_url":  baseURL,
		"final_url": finalURL,
		"endpoint":  "/chat/completions",
		"construction": map[string]string{
			"rule": "If base URL ends with '/' append 'v1', otherwise append '/v1'",
		},
	})
}

// constructFinalEndpointURL constructs the final endpoint URL using the same logic as the OpenAI client
func constructFinalEndpointURL(baseURL string) string {
	if baseURL == "" {
		return "https://api.openai.com/v1/chat/completions"
	}

	// Handle trailing slash - always remove it
	if strings.HasSuffix(baseURL, "/") {
		baseURL = strings.TrimSuffix(baseURL, "/")
	}

	// Check if URL already contains /v1 - don't add another one
	if strings.Contains(baseURL, "/v1") {
		return baseURL + "/chat/completions"
	}

	// Parse the URL to analyze its structure
	if shouldAppendV1ForHandler(baseURL) {
		return baseURL + "/v1/chat/completions"
	}

	return baseURL + "/chat/completions"
}

// shouldAppendV1ForHandler determines if /v1 should be appended based on URL structure
func shouldAppendV1ForHandler(baseURL string) bool {
	// For standard OpenAI API format (api.openai.com), we should append /v1
	if strings.Contains(strings.ToLower(baseURL), "api.openai.com") {
		return true
	}

	// Extract the path part after the domain
	parts := strings.Split(baseURL, "/")
	if len(parts) < 4 { // protocol://domain only, no path
		return true // Simple domain, likely needs /v1
	}

	// Count meaningful path segments (ignore empty strings)
	pathSegments := 0
	for i := 3; i < len(parts); i++ { // Start from index 3 to skip protocol://domain
		if parts[i] != "" {
			pathSegments++
		}
	}

	// If URL has 2+ path segments (like /api/openrouter), it's likely a proxy service
	// that has its own routing and doesn't need /v1
	if pathSegments >= 2 {
		return false
	}

	// Single path segment or simple domain - likely needs /v1
	return true
}

// ProxyTestRequest 代理测试请求结构
type ProxyTestRequest struct {
	ProxyConfig *struct {
		Type     string `json:"type"`     // "http" or "socks5"
		Address  string `json:"address"`  // 代理地址
		Username string `json:"username"` // SOCKS5用户名（可选）
		Password string `json:"password"` // SOCKS5密码（可选）
	} `json:"proxy_config"`
	TestURL   string `json:"test_url"`   // 测试URL
	IgnoreSSL bool   `json:"ignore_ssl"` // 是否忽略SSL验证
}

// ProxyTestResponse 代理测试响应结构
type ProxyTestResponse struct {
	Success  bool   `json:"success"`
	Duration int64  `json:"duration"` // 响应时间（毫秒）
	IP       string `json:"ip"`       // 获取到的IP地址
	Error    string `json:"error,omitempty"`
}

// TestProxy 测试代理连接 - POST /admin/test-proxy
func (h *ConfigHandler) TestProxy(c *gin.Context) {
	var req ProxyTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid proxy test request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	if req.ProxyConfig == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Proxy configuration is required",
		})
		return
	}

	if req.TestURL == "" {
		req.TestURL = "https://httpbin.org/ip"
	}

	h.logger.WithFields(logrus.Fields{
		"proxy_type":    req.ProxyConfig.Type,
		"proxy_address": req.ProxyConfig.Address,
		"test_url":      req.TestURL,
		"ignore_ssl":    req.IgnoreSSL,
	}).Info("Testing proxy connection")

	// 创建代理配置
	proxyConfig := &client.ProxyConfig{
		Enabled:   true,
		Type:      req.ProxyConfig.Type,
		IgnoreSSL: req.IgnoreSSL,
	}

	switch req.ProxyConfig.Type {
	case "http":
		proxyConfig.HTTPProxy = req.ProxyConfig.Address
	case "socks5":
		proxyConfig.SOCKS5Proxy = req.ProxyConfig.Address
		proxyConfig.SOCKS5ProxyUser = req.ProxyConfig.Username
		proxyConfig.SOCKS5ProxyPassword = req.ProxyConfig.Password
	}

	// 测试代理连接
	result := h.testProxyConnection(proxyConfig, req.TestURL)

	if result.Success {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusBadRequest, result)
	}
}

// testProxyConnection 测试代理连接
func (h *ConfigHandler) testProxyConnection(proxyConfig *client.ProxyConfig, testURL string) ProxyTestResponse {
	startTime := time.Now()

	// 使用代理配置构建HTTP传输
	transport := client.BuildHTTPTransport(proxyConfig)

	// 创建HTTP客户端
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// 发送请求
	resp, err := httpClient.Get(testURL)
	if err != nil {
		h.logger.WithError(err).Error("Proxy test request failed")
		return ProxyTestResponse{
			Success: false,
			Error:   fmt.Sprintf("请求失败: %v", err),
		}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			h.logger.WithError(err).Error("Failed to close response body")
		}
	}()

	duration := time.Since(startTime).Milliseconds()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.WithError(err).Error("Failed to read response body")
		return ProxyTestResponse{
			Success:  false,
			Duration: duration,
			Error:    fmt.Sprintf("读取响应失败: %v", err),
		}
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		h.logger.WithField("status_code", resp.StatusCode).Error("Non-OK response status")
		return ProxyTestResponse{
			Success:  false,
			Duration: duration,
			Error:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		}
	}

	// 尝试解析IP地址（如果是httpbin.org/ip的话）
	var ipResponse struct {
		Origin string `json:"origin"`
	}

	var detectedIP string
	if err := json.Unmarshal(body, &ipResponse); err == nil && ipResponse.Origin != "" {
		detectedIP = ipResponse.Origin
	} else {
		// 如果不是JSON格式，尝试提取IP
		bodyStr := string(body)
		if strings.Contains(bodyStr, ".") {
			detectedIP = strings.TrimSpace(bodyStr)
		}
	}

	h.logger.WithFields(logrus.Fields{
		"duration":    duration,
		"detected_ip": detectedIP,
		"status":      resp.StatusCode,
	}).Info("Proxy test completed successfully")

	return ProxyTestResponse{
		Success:  true,
		Duration: duration,
		IP:       detectedIP,
	}
}
