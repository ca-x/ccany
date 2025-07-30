package toolmapping

import "sync"

// customMappings stores dynamic custom tool name mappings
// It is protected by mapMutex to ensure concurrent-safe access.
var (
	customMappings = make(map[string]string)
	mapMutex       sync.RWMutex
)

// MapOpenAIToClaudeName maps OpenAI tool names to Claude tool names
func MapOpenAIToClaudeName(openaiName string) string {
	// Map OpenAI tool names to Claude-compatible tool names
	switch openaiName {
	case "python":
		return "str_replace_editor"
	case "bash":
		return "bash"
	case "str_replace_based_edit_tool":
		return "str_replace_editor"
	case "file_search":
		return "str_replace_editor"
	default:
		// Return the original name if no mapping is needed
		return openaiName
	}
}

// MapClaudeToOpenAIName maps Claude tool names to OpenAI tool names
func MapClaudeToOpenAIName(claudeName string) string {
	// Map Claude tool names to OpenAI-compatible tool names
	switch claudeName {
	case "str_replace_editor":
		return "str_replace_based_edit_tool"
	case "bash":
		return "bash"
	default:
		// Return the original name if no mapping is needed
		return claudeName
	}
}

// MapCustomToClaudeName maps custom tool names to Claude tool names
func MapCustomToClaudeName(customName string) string {
	mapMutex.RLock()
	defer mapMutex.RUnlock()

	if claudeName, exists := customMappings[customName]; exists {
		return claudeName
	}

	// Return the original name if no custom mapping exists
	return customName
}

// AddCustomMapping adds a custom tool name mapping
func AddCustomMapping(customName, claudeName string) {
	mapMutex.Lock()
	defer mapMutex.Unlock()

	customMappings[customName] = claudeName
}
