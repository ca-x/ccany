# Native Gemini API Support Setup Guide

This guide explains how to configure ccany to use Google's native Gemini API instead of the OpenAI-compatible endpoint.

## Overview

ccany now supports two ways to use Gemini models:

1. **OpenAI-Compatible Endpoint** (existing): Uses Gemini's OpenAI-compatible API
2. **Native Gemini API** (new): Uses Google's native Gemini API directly

The native API provides better performance, more accurate tool calling, and access to Gemini-specific features.

## Configuration

### 1. Get Gemini API Key

1. Visit [Google AI Studio](https://aistudio.google.com/app/apikey)
2. Create a new API key
3. Copy the key (starts with `AIza...`)

### 2. Configure ccany

#### Option A: Using Web Interface

1. Open ccany setup page: `http://localhost:8082/setup`
2. Add a new channel configuration:
   - **Name**: `gemini-native`
   - **Base URL**: `https://generativelanguage.googleapis.com/v1beta/models`
   - **API Key**: Your Gemini API key
   - **Model**: `gemini-1.5-flash` or `gemini-1.5-pro`

#### Option B: Using Configuration File

Add to your ccany configuration:

```yaml
channels:
  gemini-native:
    base_url: "https://generativelanguage.googleapis.com/v1beta/models"
    api_key: "AIza..."  # Your Gemini API key
    models:
      - "gemini-1.5-flash"
      - "gemini-1.5-pro"
      - "gemini-2.0-flash-exp"
```

## Usage

### Claude Code Integration

When using Claude Code, ccany will automatically detect Gemini endpoints and use the native API:

```python
import anthropic

client = anthropic.Anthropic(
    api_key="your-ccany-key",
    base_url="http://localhost:8082"
)

response = client.messages.create(
    model="gemini-1.5-flash",
    max_tokens=1000,
    messages=[{"role": "user", "content": "Hello!"}],
    tools=[
        {
            "name": "web_search",
            "description": "Search the web",
            "input_schema": {
                "type": "object",
                "properties": {
                    "query": {"type": "string"}
                },
                "required": ["query"]
            }
        }
    ]
)
```

### Direct API Usage

```bash
curl -X POST http://localhost:8082/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-ccany-key" \
  -d '{
    "model": "gemini-1.5-flash",
    "max_tokens": 1000,
    "messages": [
      {"role": "user", "content": "What is the weather like?"}
    ],
    "tools": [
      {
        "name": "get_weather",
        "description": "Get current weather",
        "input_schema": {
          "type": "object",
          "properties": {
            "location": {"type": "string"}
          },
          "required": ["location"]
        }
      }
    ]
  }'
```

## Features

### ✅ Supported Features

- **All Claude Code tools**: web_search, file operations, bash commands, etc.
- **Tool calling**: Full function calling support with proper schema conversion
- **Streaming responses**: Real-time response streaming
- **System prompts**: Converted to Gemini's SystemInstruction format
- **Message history**: Full conversation context preservation
- **Error handling**: Comprehensive error handling and logging

### 🔄 Automatic Conversions

ccany automatically handles:

- **Schema sanitization**: Removes Gemini-incompatible schema properties
- **Message format**: Converts Claude messages to Gemini Contents
- **Tool definitions**: Transforms Claude tools to Gemini FunctionDeclarations
- **Response format**: Converts Gemini responses back to Claude format

## Troubleshooting

### Common Issues

1. **"Invalid API key"**
   - Verify your Gemini API key is correct
   - Ensure the key has proper permissions

2. **"Schema validation failed"**
   - Check tool schema definitions
   - ccany automatically sanitizes schemas, but complex nested schemas may need adjustment

3. **"Model not found"**
   - Verify the model name is correct
   - Supported models: `gemini-1.5-flash`, `gemini-1.5-pro`, `gemini-2.0-flash-exp`

### Debug Logging

Enable debug logging to see detailed request/response information:

```bash
LOG_LEVEL=debug ./ccany
```

Look for log entries containing:
- `"Processing Claude Code compatible request"`
- `"Detected Gemini provider"`
- `"Converting Claude request to Gemini format"`

## Performance Benefits

Native Gemini API provides:

- **Faster responses**: Direct API communication without OpenAI compatibility layer
- **Better tool calling**: More accurate function calling with proper schema handling
- **Lower latency**: Reduced request processing overhead
- **Full feature access**: Access to all Gemini-specific capabilities

## Migration from OpenAI-Compatible Endpoint

To migrate existing Gemini configurations:

1. **Update base URL**: Change from `/v1beta/openai/` to `/v1beta/models`
2. **Keep API key**: Same Gemini API key works for both endpoints
3. **Test thoroughly**: Verify tool calling and response formats work as expected

The native API is backward compatible with existing Claude Code integrations.