// Package utils provides utility functions for testing and development
package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// OllamaModel represents a model available in Ollama
type OllamaModel struct {
	Name       string       `json:"name"`
	ModifiedAt time.Time    `json:"modified_at"`
	Size       int64        `json:"size"`
	Digest     string       `json:"digest"`
	Details    ModelDetails `json:"details"`
}

// ModelDetails contains detailed information about a model
type ModelDetails struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// OllamaModelsResponse represents the response from Ollama's models API
type OllamaModelsResponse struct {
	Models []OllamaModel `json:"models"`
}

// PreferredModels defines the order of preference for testing models
// These models are known to support tools/function calling
var PreferredModels = []string{
	"qwen2.5:14b",
	"qwen2.5:7b",
	"qwen2.5:3b",
	"qwen2.5:1.5b",
	"llama3.1:8b",
	"llama3.1:7b",
	"llama3.2:3b",
	"llama3.2:1b",
	"mistral:7b",
	"phi3:3.8b",
	"phi3:mini",
}

// ToolCapableModels contains model families known to support tools
var ToolCapableModels = map[string]bool{
	"qwen":      true,
	"qwen2":     true,
	"qwen2.5":   true,
	"llama3":    true,
	"llama3.1":  true,
	"llama3.2":  true,
	"mistral":   true,
	"phi3":      true,
	"gemma":     false, // Most Gemma models don't support tools well
	"codellama": false, // Code-focused, limited tool support
}

// IsOllamaRunning checks if Ollama server is running and accessible
func IsOllamaRunning() bool {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK
}

// GetAvailableModels retrieves the list of models available in Ollama
func GetAvailableModels() ([]OllamaModel, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ollama: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama API returned status %d", resp.StatusCode)
	}

	var response OllamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	return response.Models, nil
}

// IsModelToolCapable checks if a model supports tools based on its family
func IsModelToolCapable(modelName string) bool {
	// Extract the model family from the full model name
	// e.g., "qwen2.5:7b" -> "qwen2.5", "llama3.1:8b" -> "llama3.1"
	parts := strings.Split(modelName, ":")
	if len(parts) == 0 {
		return false
	}

	family := parts[0]

	// Check exact match first
	if capable, exists := ToolCapableModels[family]; exists {
		return capable
	}

	// Check for partial matches (e.g., "qwen2.5" should match "qwen")
	for knownFamily, capable := range ToolCapableModels {
		if strings.HasPrefix(family, knownFamily) && capable {
			return true
		}
	}

	return false
}

// FindBestAvailableModel finds the best available model for testing
// Returns the model name and whether it supports tools
func FindBestAvailableModel() (string, bool, error) {
	models, err := GetAvailableModels()
	if err != nil {
		return "", false, err
	}

	// Create a map of available models for quick lookup
	availableModels := make(map[string]bool)
	for _, model := range models {
		availableModels[model.Name] = true
	}

	// Check preferred models in order
	for _, preferredModel := range PreferredModels {
		if availableModels[preferredModel] {
			toolCapable := IsModelToolCapable(preferredModel)
			return preferredModel, toolCapable, nil
		}
	}

	// If no preferred model is found, try to find any tool-capable model
	for _, model := range models {
		if IsModelToolCapable(model.Name) {
			return model.Name, true, nil
		}
	}

	// If no tool-capable model found, return the first available model
	if len(models) > 0 {
		return models[0].Name, false, nil
	}

	return "", false, fmt.Errorf("no models available in Ollama")
}

// SkipIfOllamaNotRunning skips the test if Ollama is not running
func SkipIfOllamaNotRunning(t *testing.T) {
	if !IsOllamaRunning() {
		t.Skip("Skipping test: Ollama server is not running")
	}
}

// RequireOllamaWithTools skips the test if Ollama is not running or no tool-capable models are available
func RequireOllamaWithTools(t *testing.T) string {
	SkipIfOllamaNotRunning(t)

	modelName, supportsTools, err := FindBestAvailableModel()
	if err != nil {
		t.Skipf("Skipping test: failed to find available model: %v", err)
	}

	if !supportsTools {
		t.Skipf("Skipping test: no tool-capable models available (found: %s)", modelName)
	}

	return modelName
}

// RequireOllama skips the test if Ollama is not running but returns any available model
func RequireOllama(t *testing.T) string {
	SkipIfOllamaNotRunning(t)

	modelName, _, err := FindBestAvailableModel()
	if err != nil {
		t.Skipf("Skipping test: failed to find available model: %v", err)
	}

	return modelName
}
