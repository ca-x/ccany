# PR Title: Feat: Add Native Gemini Support & Fix Claude Tool Call Hanging

## Summary

This pull request introduces a dedicated, native Gemini API integration for `ccany` and resolves a critical bug that caused the Claude Code client to hang during tool call operations.

The system now correctly transforms Claude API requests to the native Gemini REST API format, including robust sanitation of complex tool schemas. This bypasses the flawed "OpenAI-Compatible" Gemini endpoint, ensuring reliable tool execution. Additionally, the original Claude-to-OpenAI converter has been fixed to prevent it from forcing tool usage, which was the root cause of the client hanging.

## 1. The Problem

When using `ccany` as a proxy between the Claude Code client and a backend LLM (either OpenAI-compatible or Gemini), two major issues were observed:

1. **Client Hanging (Critical Bug):** When a user issued a command that required a tool (e.g., "what is the weather?"), the `ccany` server would process the request, but the Claude Code client in the terminal would hang indefinitely, never showing a response.
2. **Gemini Tool Failure:** When `ccany` was pointed at a Gemini backend, tool calls would fail with a `400 Bad Request` error, indicating a severe schema mismatch.

## 2. Root Cause Analysis

Our investigation revealed two independent root causes:

1. **Incorrect Tool Choice Logic (Hanging Bug):** The primary cause of the client hanging was in `internal/converter/request.go`. The logic **incorrectly forced `tool_choice: "required"`** on almost all requests that included tools. This forced the backend model to *only* return a tool call, skipping the preliminary text response (e.g., "Okay, let me check that for you."). The Claude Code client's UI requires this initial text block to proceed and would hang waiting for it.

2. **Schema Incompatibility (Gemini Failure):** The `ccany` proxy was designed to convert all requests to the OpenAI format. We discovered that:
   - Gemini's "OpenAI-Compatible" endpoint is unreliable and does not correctly handle the standard OpenAI `tool_calls` schema.
   - Claude Code's tool definitions use a complex JSON Schema format that is fundamentally incompatible with the much stricter, Protobuf-style schema expected by the native Gemini API. A simple conversion was not possible.

## 3. The Solution: Exact Changes Implemented

To resolve these issues, we implemented a two-pronged solution: we fixed the core OpenAI conversion logic and added a new, dedicated path for native Gemini support.

### Part A: Bug Fix for Core OpenAI Converter

These changes fix the client hanging issue for all OpenAI-compatible backends.

1. **`internal/converter/request.go`**
   - **Modified `ConvertClaudeToOpenAI`:** The logic that forced `tool_choice: "required"` was removed. The converter now correctly respects the `tool_choice` value (`"auto"`, `"any"`, etc.) sent by the original Claude Code client, allowing the model to respond with text before a tool call.

2. **`internal/converter/response.go`**
   - **Replaced `convertMessageToClaudeContent`:** The function was completely rewritten to be simpler and more robust. It now correctly prioritizes adding the `text` content block to the response *before* adding any `tool_use` blocks, ensuring the client UI never hangs.

### Part B: New Feature - Native Gemini Provider Integration

This adds a new, parallel execution path to `ccany` for directly communicating with the native Gemini API.

1. **`internal/models/gemini.go` (New File)**
   - Created Go structs (`GeminiRequest`, `GeminiContent`, `GeminiPart`, `GeminiFunctionCall`, etc.) that precisely match the JSON schema of the **native Gemini REST API**. This schema was validated against the official Google Gemini SDK documentation and our successful `curl` tests.

2. **`internal/client/gemini.go` (New File)**
   - Created a new, dedicated `GeminiClient`.
   - This client correctly constructs the native Gemini URL format: `.../models/{model}:generateContent`.
   - It correctly handles authentication by sending the API key as a **`?key=` query parameter**, not a `Bearer` token.

3. **`internal/converter/gemini_converter.go` (New File)**
   - Created a new `GeminiConverter` to handle the `Claude <-> Native Gemini` schema transformation.
   - **Implemented `ConvertFromClaude`:** This function is the core of the Gemini fix. It contains a robust, recursive **`sanitizeSchema` function** that rebuilds the complex JSON Schema from Claude's tools into the strict, Protobuf-style schema that Gemini requires. It correctly maps all message types, system prompts, and tool configurations.

4. **`internal/handlers/enhanced_messages.go` (Modified)**
   - **Added Routing Logic:** The main `CreateMessage` handler was updated. It now inspects the configured `OpenAIBaseURL`.
     - If it detects a Gemini URL (`generativelanguage.googleapis.com`), it routes the request through the new `GeminiConverter` and `GeminiClient`.
     - Otherwise, it uses the existing (and now fixed) OpenAI path.

5. **Configuration (Implied Change)**
   - The system now supports pointing the `OpenAIBaseURL` in the configuration to either an OpenAI-compatible endpoint or the native Gemini endpoint, and the routing logic will handle it correctly.

---

This PR makes `ccany` a truly multi-provider proxy, fixing a critical usability bug and adding robust, native support for one of the most popular alternative LLM backends.

## Potential Future Errors to Watch For

Based on our work, here are the exact errors you should look out for:

### Category 1: Gemini API Errors (External)
- **401/403** - Invalid Authentication: API key issues
- **429** - Quota Exceeded: Rate limiting or daily limits
- **400** - Invalid Argument: New Claude tools with unsupported schema features

### Category 2: ccany Conversion Errors (Internal)
- **Panic During Schema Conversion**: Unexpected tool schema structures
- **Panic During Response Conversion**: New Gemini response part types

### Category 3: Operational Errors
- **Request Timeouts**: Complex requests taking too long

The most critical error to monitor is **400 Bad Request** from Gemini, which would indicate that Claude Code has introduced a new tool with a schema feature our `sanitizeSchema` function doesn't handle. This would require updating the sanitization logic.
