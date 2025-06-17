//go:build openai || all

package models

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/kris-hansen/comanda/utils/fileutil"
	openai "github.com/sashabaranov/go-openai"
)

// OpenAIProvider handles OpenAI family of models
type OpenAIProvider struct {
	apiKey  string
	config  ModelConfig
	verbose bool
}

// NewOpenAIProvider creates a new OpenAI provider instance
func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		config: ModelConfig{
			Temperature:         0.7,
			MaxTokens:           2000,
			MaxCompletionTokens: 2000,
			TopP:                1.0,
		},
	}
}

// Name returns the provider name
func (o *OpenAIProvider) Name() string {
	return "openai"
}

// debugf prints debug information if verbose mode is enabled
func (o *OpenAIProvider) debugf(format string, args ...interface{}) {
	if o.verbose {
		fmt.Printf("[DEBUG][OpenAI] "+format+"\n", args...)
	}
}

// SupportsModel checks if the given model name is supported by OpenAI
func (o *OpenAIProvider) SupportsModel(modelName string) bool {
	o.debugf("Checking if model is supported: %s", modelName)
	modelName = strings.ToLower(modelName)

	// Accept any model name that starts with our known prefixes
	validPrefixes := []string{
		"gpt-",     // Standard GPT models
		"o1-",      // Covers o1-pro, o1-preview etc
		"o3-",      // Support for o3 models
		"o4-",      // Support for o4-mini series
		"gpt-4o",   // Support for gpt-4o variants
		"gpt-4.1-", // Support for gpt-4.1 series (mini, nano)
	}

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(modelName, prefix) {
			o.debugf("Model %s is supported (matches prefix %s)", modelName, prefix)
			return true
		}
	}

	o.debugf("Model %s is not supported (no matching prefix)", modelName)
	return false
}

// Configure sets up the provider with necessary credentials
func (o *OpenAIProvider) Configure(apiKey string) error {
	o.debugf("Configuring OpenAI provider")
	if apiKey == "" {
		return fmt.Errorf("API key is required for OpenAI provider")
	}
	o.apiKey = apiKey
	o.debugf("API key configured successfully")
	return nil
}

// isNewModelSeries checks if the model is part of the newer series (4o, o1, o3, o4)
func (o *OpenAIProvider) isNewModelSeries(modelName string) bool {
	modelName = strings.ToLower(modelName)
	return strings.Contains(modelName, "4o") ||
		strings.HasPrefix(modelName, "o1-") ||
		strings.HasPrefix(modelName, "o3-") ||
		strings.HasPrefix(modelName, "o4-") // Support for o4-mini series
}

// createChatCompletionRequest creates a ChatCompletionRequest with the appropriate parameters
func (o *OpenAIProvider) createChatCompletionRequest(modelName string, messages []openai.ChatCompletionMessage) openai.ChatCompletionRequest {
	req := openai.ChatCompletionRequest{
		Model:    modelName,
		Messages: messages,
	}

	if o.isNewModelSeries(modelName) {
		// New model series (4o and o1) have fixed parameters
		req.MaxCompletionTokens = o.config.MaxCompletionTokens
		req.Temperature = 1.0
		req.TopP = 1.0
		req.PresencePenalty = 0.0
		req.FrequencyPenalty = 0.0
		o.debugf("Using fixed parameters for new model series: Temperature=1.0, TopP=1.0, PresencePenalty=0.0, FrequencyPenalty=0.0")
	} else {
		// Legacy models use configurable parameters
		req.MaxTokens = o.config.MaxTokens
		req.Temperature = float32(o.config.Temperature)
		req.TopP = float32(o.config.TopP)
		o.debugf("Using configured parameters for legacy model: Temperature=%.2f, TopP=%.2f", o.config.Temperature, o.config.TopP)
	}

	return req
}

// SendPrompt sends a prompt to the specified model and returns the response
func (o *OpenAIProvider) SendPrompt(modelName string, prompt string) (string, error) {
	o.debugf("Preparing to send prompt to model: %s", modelName)
	o.debugf("Prompt length: %d characters", len(prompt))

	if o.apiKey == "" {
		return "", fmt.Errorf("OpenAI provider not configured: missing API key")
	}

	if !o.SupportsModel(modelName) {
		return "", fmt.Errorf("invalid OpenAI model: %s", modelName)
	}

	o.debugf("Model validation passed, preparing API call")

	client := openai.NewClient(o.apiKey)

	// Check if this is a vision input by looking for base64 image data
	if strings.HasPrefix(modelName, "gpt-4") && strings.Contains(prompt, ";base64,") {
		return o.handleVisionPrompt(client, prompt, modelName)
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	req := o.createChatCompletionRequest(modelName, messages)
	resp, err := client.CreateChatCompletion(context.Background(), req)

	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from OpenAI")
	}

	response := resp.Choices[0].Message.Content
	o.debugf("API call completed, response length: %d characters", len(response))

	return response, nil
}

// SendPromptWithFile sends a prompt along with a file to the specified model and returns the response
func (o *OpenAIProvider) SendPromptWithFile(modelName string, prompt string, file FileInput) (string, error) {
	o.debugf("Preparing to send prompt with file to model: %s", modelName)
	o.debugf("File path: %s", file.Path)

	if o.apiKey == "" {
		return "", fmt.Errorf("OpenAI provider not configured: missing API key")
	}

	if !o.SupportsModel(modelName) {
		return "", fmt.Errorf("invalid OpenAI model: %s", modelName)
	}

	// Read the file content with size check
	fileData, err := fileutil.SafeReadFile(file.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	client := openai.NewClient(o.apiKey)

	// For GPT-4 Vision, handle image files
	if strings.HasPrefix(modelName, "gpt-4") && strings.HasPrefix(file.MimeType, "image/") {
		return o.handleFileAsVision(client, prompt, fileData, file.MimeType, modelName)
	}

	// For other files, include the content as part of the prompt
	fileContent := string(fileData)
	combinedPrompt := fmt.Sprintf("File content:\n%s\n\nUser prompt: %s", fileContent, prompt)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: combinedPrompt,
		},
	}

	req := o.createChatCompletionRequest(modelName, messages)
	resp, err := client.CreateChatCompletion(context.Background(), req)

	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from OpenAI")
	}

	response := resp.Choices[0].Message.Content
	o.debugf("API call completed, response length: %d characters", len(response))

	return response, nil
}

// handleFileAsVision processes a file as a vision model request
func (o *OpenAIProvider) handleFileAsVision(client *openai.Client, prompt string, fileData []byte, mimeType string, modelName string) (string, error) {
	// Convert file data to base64 string with proper data URI prefix
	base64Data := fmt.Sprintf("data:%s;base64,%s", mimeType, string(fileData))

	// Create the message content with text and image parts
	content := []openai.ChatMessagePart{
		{
			Type: openai.ChatMessagePartTypeText,
			Text: prompt,
		},
		{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL: base64Data,
			},
		},
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:         openai.ChatMessageRoleUser,
			MultiContent: content,
		},
	}

	req := o.createChatCompletionRequest(modelName, messages)
	resp, err := client.CreateChatCompletion(context.Background(), req)

	if err != nil {
		return "", fmt.Errorf("OpenAI Vision API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from OpenAI Vision")
	}

	return resp.Choices[0].Message.Content, nil
}

// handleVisionPrompt processes a vision model request with image data
func (o *OpenAIProvider) handleVisionPrompt(client *openai.Client, prompt string, modelName string) (string, error) {
	// Split the prompt into text and base64 image data
	parts := strings.Split(prompt, "Action: ")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid vision prompt format")
	}

	// Extract image data from the input section
	inputParts := strings.Split(parts[0], "Input:\n")
	if len(inputParts) != 2 {
		return "", fmt.Errorf("invalid input format in vision prompt")
	}

	imageData := strings.TrimSpace(inputParts[1])
	action := strings.TrimSpace(parts[1])

	o.debugf("Vision prompt image data length: %d", len(imageData))
	o.debugf("Vision prompt action length: %d", len(action))

	// Check if image data is properly formatted
	if !strings.HasPrefix(imageData, "data:image/") && !strings.Contains(imageData, ";base64,") {
		imageData = fmt.Sprintf("data:image/png;base64,%s", imageData)
	}

	// Create the message content with text and image parts
	content := []openai.ChatMessagePart{
		{
			Type: openai.ChatMessagePartTypeText,
			Text: action,
		},
		{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL: imageData,
			},
		},
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:         openai.ChatMessageRoleUser,
			MultiContent: content,
		},
	}

	req := o.createChatCompletionRequest(modelName, messages)
	resp, err := client.CreateChatCompletion(context.Background(), req)

	if err != nil {
		return "", fmt.Errorf("OpenAI Vision API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from OpenAI Vision")
	}

	return resp.Choices[0].Message.Content, nil
}

// ValidateModel checks if the specific OpenAI model variant is valid
func (o *OpenAIProvider) ValidateModel(modelName string) bool {
	return o.SupportsModel(modelName)
}

// SetConfig updates the provider configuration
func (o *OpenAIProvider) SetConfig(config ModelConfig) {
	o.debugf("Updating provider configuration")
	o.debugf("Old config: Temperature=%.2f, MaxTokens=%d, MaxCompletionTokens=%d, TopP=%.2f",
		o.config.Temperature, o.config.MaxTokens, o.config.MaxCompletionTokens, o.config.TopP)
	o.config = config
	o.debugf("New config: Temperature=%.2f, MaxTokens=%d, MaxCompletionTokens=%d, TopP=%.2f",
		o.config.Temperature, o.config.MaxTokens, o.config.MaxCompletionTokens, o.config.TopP)
}

// GetConfig returns the current provider configuration
func (o *OpenAIProvider) GetConfig() ModelConfig {
	return o.config
}

// SetVerbose enables or disables verbose mode
func (o *OpenAIProvider) SetVerbose(verbose bool) {
	o.verbose = verbose
}

// ListModels returns the list of available models from OpenAI
func (o *OpenAIProvider) ListModels() ([]string, error) {
	return ListModelsForProvider(o.Name(), o.apiKey)
}

// prepareResponsesRequestBody prepares the request body for the Responses API
func (o *OpenAIProvider) prepareResponsesRequestBody(config ResponsesConfig) (map[string]interface{}, error) {
	// Build the request body
	requestBody := map[string]interface{}{
		"model": config.Model,
		"input": config.Input,
	}

	// Add stream parameter if specified
	if config.Stream {
		requestBody["stream"] = true
	}

	// Add optional parameters if provided
	if config.Instructions != "" {
		requestBody["instructions"] = config.Instructions
	}

	if config.PreviousResponseID != "" {
		requestBody["previous_response_id"] = config.PreviousResponseID
	}

	if config.MaxOutputTokens > 0 {
		requestBody["max_output_tokens"] = config.MaxOutputTokens
	}

	if config.Temperature > 0 {
		requestBody["temperature"] = config.Temperature
	}

	if config.TopP > 0 {
		requestBody["top_p"] = config.TopP
	}

	if len(config.Tools) > 0 {
		// Format tools correctly for the API
		var formattedTools []map[string]interface{}

		// Debug the tools
		toolsBytes, _ := json.Marshal(config.Tools)
		o.debugf("Tools: %s", string(toolsBytes))

		// Process each tool
		for _, toolMap := range config.Tools {
			o.debugf("Processing tool: %v", toolMap)

			// Get the tool type
			toolTypeRaw, ok := toolMap["type"]
			if !ok {
				o.debugf("Tool has no type: %v", toolMap)
				continue
			}

			toolType, ok := toolTypeRaw.(string)
			if !ok {
				o.debugf("Tool type is not a string: %v", toolTypeRaw)
				continue
			}

			// Process based on tool type
			if toolType == "function" {
				functionRaw, ok := toolMap["function"]
				if !ok {
					o.debugf("Function tool has no function field: %v", toolMap)
					continue
				}

				function, ok := functionRaw.(map[string]interface{})
				if !ok {
					o.debugf("Function field is not a map: %v", functionRaw)
					continue
				}

				// Extract the name from the function object
				name, ok := function["name"].(string)
				if !ok {
					o.debugf("Function has no name field: %v", function)
					continue
				}

				// Add the formatted function tool with name at the top level
				formattedTools = append(formattedTools, map[string]interface{}{
					"type":     "function",
					"name":     name,
					"function": function,
				})
				o.debugf("Added function tool: %s", name)
			} else {
				// Other tool types (like web_search)
				formattedTools = append(formattedTools, toolMap)
				o.debugf("Added tool of type %s", toolType)
			}
		}

		// Add the formatted tools to the request
		requestBody["tools"] = formattedTools
		o.debugf("Final formatted tools: %v", formattedTools)
	}

	if config.ResponseFormat != nil {
		requestBody["text"] = config.ResponseFormat
	}

	return requestBody, nil
}

// SendPromptWithResponses sends a prompt using the OpenAI Responses API
func (o *OpenAIProvider) SendPromptWithResponses(config ResponsesConfig) (string, error) {
	o.debugf("Preparing to send prompt using Responses API with model: %s", config.Model)

	if o.apiKey == "" {
		return "", fmt.Errorf("OpenAI provider not configured: missing API key")
	}

	if !o.SupportsModel(config.Model) {
		return "", fmt.Errorf("invalid OpenAI model: %s", config.Model)
	}

	// Prepare request body
	requestBody, err := o.prepareResponsesRequestBody(config)
	if err != nil {
		return "", fmt.Errorf("failed to prepare request body: %w", err)
	}

	// Convert request body to JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request with context for timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/responses", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.apiKey))

	// Implement retry logic
	maxRetries := 3
	var lastErr error
	var responseData map[string]interface{}

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoffDuration := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			time.Sleep(backoffDuration)
			o.debugf("Retrying request (attempt %d/%d) after %v", attempt+1, maxRetries, backoffDuration)
		}

		// Send request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send HTTP request (attempt %d/%d): %w", attempt+1, maxRetries, err)
			continue // Retry
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body (attempt %d/%d): %w", attempt+1, maxRetries, err)
			continue // Retry
		}

		// Check for error status code
		if resp.StatusCode != http.StatusOK {
			// Don't retry on 4xx errors (client errors)
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				return "", fmt.Errorf("OpenAI API error: %s (status code: %d)", string(body), resp.StatusCode)
			}

			lastErr = fmt.Errorf("OpenAI API error (attempt %d/%d): %s (status code: %d)",
				attempt+1, maxRetries, string(body), resp.StatusCode)
			continue // Retry on 5xx errors
		}

		// Parse response
		if err := json.Unmarshal(body, &responseData); err != nil {
			lastErr = fmt.Errorf("failed to parse response body (attempt %d/%d): %w", attempt+1, maxRetries, err)
			continue // Retry
		}

		// If we get here, the request was successful
		break
	}

	// Check if all retries failed
	if responseData == nil {
		return "", fmt.Errorf("all retry attempts failed: %w", lastErr)
	}

	// Extract output text
	output, err := o.extractOutputText(responseData)
	if err != nil {
		return "", fmt.Errorf("failed to extract output text: %w", err)
	}

	o.debugf("API call completed, response length: %d characters", len(output))

	return output, nil
}

// SendPromptWithResponsesStream sends a prompt using the OpenAI Responses API with streaming
func (o *OpenAIProvider) SendPromptWithResponsesStream(config ResponsesConfig, handler ResponsesStreamHandler) error {
	o.debugf("Preparing to send prompt using Responses API with streaming for model: %s", config.Model)

	if o.apiKey == "" {
		return fmt.Errorf("OpenAI provider not configured: missing API key")
	}

	if !o.SupportsModel(config.Model) {
		return fmt.Errorf("invalid OpenAI model: %s", config.Model)
	}

	// Force streaming to be enabled
	config.Stream = true

	// Prepare request body
	requestBody, err := o.prepareResponsesRequestBody(config)
	if err != nil {
		return fmt.Errorf("failed to prepare request body: %w", err)
	}

	// Convert request body to JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request with context for timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/responses", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.apiKey))
	req.Header.Set("Accept", "text/event-stream")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenAI API error: %s (status code: %d)", string(body), resp.StatusCode)
	}

	// Process the stream using a scanner
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Skip "data: " prefix
		line = strings.TrimPrefix(line, "data: ")

		// Skip "[DONE]" message
		if line == "[DONE]" {
			break
		}

		// Parse the event
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			handler.OnError(fmt.Errorf("failed to parse event: %w", err))
			continue
		}

		// Process based on event type
		eventType, ok := event["type"].(string)
		if !ok {
			continue
		}

		switch eventType {
		case "response.created":
			if resp, ok := event["response"].(map[string]interface{}); ok {
				handler.OnResponseCreated(resp)
			}
		case "response.in_progress":
			if resp, ok := event["response"].(map[string]interface{}); ok {
				handler.OnResponseInProgress(resp)
			}
		case "response.output_item.added":
			if index, ok := event["output_index"].(float64); ok {
				if item, ok := event["item"].(map[string]interface{}); ok {
					handler.OnOutputItemAdded(int(index), item)
				}
			}
		case "response.output_text.delta":
			itemID, _ := event["item_id"].(string)
			index, _ := event["output_index"].(float64)
			contentIndex, _ := event["content_index"].(float64)
			delta, _ := event["delta"].(string)
			handler.OnOutputTextDelta(itemID, int(index), int(contentIndex), delta)
		case "response.completed":
			if resp, ok := event["response"].(map[string]interface{}); ok {
				handler.OnResponseCompleted(resp)
			}
			return nil // End streaming
		case "error":
			message, _ := event["message"].(string)
			handler.OnError(fmt.Errorf("stream error: %s", message))
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream read error: %w", err)
	}

	return nil
}

// extractOutputText extracts the output text from the Responses API response
func (o *OpenAIProvider) extractOutputText(responseData map[string]interface{}) (string, error) {
	// Debug the full response
	responseBytes, _ := json.MarshalIndent(responseData, "", "  ")
	o.debugf("Full response: %s", string(responseBytes))

	// Check if output exists
	output, ok := responseData["output"]
	if !ok {
		// If no output field, check for other fields that might contain the response
		if text, ok := responseData["text"].(string); ok {
			return text, nil
		}
		if content, ok := responseData["content"].(string); ok {
			return content, nil
		}
		if message, ok := responseData["message"].(string); ok {
			return message, nil
		}

		// Return the entire response as a string if we can't find a specific field
		return string(responseBytes), nil
	}

	// Output is an array of content items
	outputArray, ok := output.([]interface{})
	if !ok {
		// If output is not an array, try to convert it to a string
		if outputStr, ok := output.(string); ok {
			return outputStr, nil
		}

		// If output is a map, try to extract text from it
		if outputMap, ok := output.(map[string]interface{}); ok {
			if text, ok := outputMap["text"].(string); ok {
				return text, nil
			}
		}

		// Return the output as a JSON string
		outputBytes, _ := json.MarshalIndent(output, "", "  ")
		return string(outputBytes), nil
	}

	// Process each output item
	var result strings.Builder
	var annotations []map[string]interface{}

	// First pass: collect all text content and annotations
	for _, item := range outputArray {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Skip web_search_call items
		if itemType, ok := itemMap["type"].(string); ok && itemType == "web_search_call" {
			o.debugf("Skipping web_search_call item")
			continue
		}

		// Check if this is a message
		if itemType, ok := itemMap["type"].(string); ok && itemType == "message" {
			// Get content array
			content, ok := itemMap["content"].([]interface{})
			if !ok {
				continue
			}

			// Process each content item
			for _, contentItem := range content {
				contentMap, ok := contentItem.(map[string]interface{})
				if !ok {
					continue
				}

				// Check if this is output_text
				if contentType, ok := contentMap["type"].(string); ok && contentType == "output_text" {
					if text, ok := contentMap["text"].(string); ok {
						result.WriteString(text)
						result.WriteString("\n")
						o.debugf("Extracted text from output_text: %d characters", len(text))

						// Collect annotations if present
						if annotationsArray, ok := contentMap["annotations"].([]interface{}); ok && len(annotationsArray) > 0 {
							o.debugf("Found %d annotations", len(annotationsArray))
							for _, anno := range annotationsArray {
								if annoMap, ok := anno.(map[string]interface{}); ok {
									annotations = append(annotations, annoMap)
								}
							}
						}
					}
				}
			}
		} else {
			// Try to extract text from the item
			if text, ok := itemMap["text"].(string); ok {
				result.WriteString(text)
				result.WriteString("\n")
				o.debugf("Extracted text from item: %d characters", len(text))
			} else if content, ok := itemMap["content"].(string); ok {
				result.WriteString(content)
				result.WriteString("\n")
				o.debugf("Extracted content from item: %d characters", len(content))
			}
		}
	}

	// If we didn't extract any text, try a more aggressive approach
	if result.Len() == 0 {
		o.debugf("No text extracted using standard approach, trying recursive extraction")
		extractedText := o.recursiveExtractText(responseData)
		if extractedText != "" {
			result.WriteString(extractedText)
			o.debugf("Extracted text using recursive approach: %d characters", len(extractedText))
		}
	}

	// If we still didn't extract any text, return the entire response as a string
	if result.Len() == 0 {
		o.debugf("No text extracted, returning entire response")
		return string(responseBytes), nil
	}

	// Add annotations as footnotes if present
	if len(annotations) > 0 {
		result.WriteString("\n\n## References\n")
		for i, anno := range annotations {
			annoType, _ := anno["type"].(string)
			if annoType == "url_citation" {
				url, _ := anno["url"].(string)
				title, _ := anno["title"].(string)
				result.WriteString(fmt.Sprintf("%d. [%s](%s)\n", i+1, title, url))
			}
		}
	}

	o.debugf("Total extracted text: %d characters", result.Len())
	return result.String(), nil
}

// recursiveExtractText recursively searches for text content in a nested structure
func (o *OpenAIProvider) recursiveExtractText(data interface{}) string {
	var result strings.Builder

	switch v := data.(type) {
	case map[string]interface{}:
		// Check for common text fields
		if text, ok := v["text"].(string); ok {
			result.WriteString(text)
			result.WriteString("\n")
			return result.String()
		}

		// Check for content field
		if content, ok := v["content"].(string); ok {
			result.WriteString(content)
			result.WriteString("\n")
			return result.String()
		}

		// Recursively search all fields
		for _, value := range v {
			text := o.recursiveExtractText(value)
			if text != "" {
				result.WriteString(text)
			}
		}
	case []interface{}:
		// Recursively search array elements
		for _, item := range v {
			text := o.recursiveExtractText(item)
			if text != "" {
				result.WriteString(text)
			}
		}
	}

	return result.String()
}

// Register the OpenAI provider on package initialization
func init() {
	factory := NewProviderFactory(
		func() Provider { return NewOpenAIProvider() },
		ProviderMetadata{
			Name:          "openai",
			Description:   "OpenAI GPT models (gpt-4, gpt-3.5-turbo, o1-, o3-, etc.)",
			Version:       "1.0.0",
			ModelPrefixes: []string{"gpt-", "o1-", "o3-"},
			Priority:      85, // High priority for GPT models
		},
	)
	RegisterProvider("openai", factory)
}
