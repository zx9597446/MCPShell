package utils

import (
	"testing"
)

func TestIsOllamaRunning(t *testing.T) {
	// This test will check if Ollama is running
	// The result will depend on whether Ollama is actually running
	running := IsOllamaRunning()
	t.Logf("Ollama running: %v", running)

	// We don't assert a specific value since it depends on the environment
	// This test is mainly to verify the function doesn't panic
}

func TestGetAvailableModels(t *testing.T) {
	if !IsOllamaRunning() {
		t.Skip("Skipping test: Ollama is not running")
	}

	models, err := GetAvailableModels()
	if err != nil {
		t.Fatalf("Failed to get available models: %v", err)
	}

	t.Logf("Found %d models", len(models))
	for _, model := range models {
		t.Logf("Model: %s (size: %d bytes)", model.Name, model.Size)
	}
}

func TestFindBestAvailableModel(t *testing.T) {
	if !IsOllamaRunning() {
		t.Skip("Skipping test: Ollama is not running")
	}

	modelName, supportsTools, err := FindBestAvailableModel()
	if err != nil {
		t.Fatalf("Failed to find best available model: %v", err)
	}

	t.Logf("Best available model: %s (supports tools: %v)", modelName, supportsTools)

	if modelName == "" {
		t.Error("Expected non-empty model name")
	}
}

func TestIsModelToolCapable(t *testing.T) {
	tests := []struct {
		model    string
		expected bool
	}{
		{"qwen2.5:7b", true},
		{"llama3.1:8b", true},
		{"mistral:7b", true},
		{"gemma:7b", false},
		{"unknown:model", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := IsModelToolCapable(tt.model)
			if result != tt.expected {
				t.Errorf("IsModelToolCapable(%s) = %v, expected %v", tt.model, result, tt.expected)
			}
		})
	}
}

func TestRequireOllamaWithTools(t *testing.T) {
	// Create a sub-test that should skip if no tool-capable models are available
	t.Run("WithOllamaAndTools", func(t *testing.T) {
		modelName := RequireOllamaWithTools(t)
		t.Logf("Got tool-capable model: %s", modelName)

		if !IsModelToolCapable(modelName) {
			t.Errorf("Model %s should be tool-capable", modelName)
		}
	})
}
