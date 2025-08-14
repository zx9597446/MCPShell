package agent

import (
	"testing"

	"github.com/inercia/MCPShell/pkg/common"
)

func TestNewModelManager(t *testing.T) {
	logger, err := common.NewLogger("", "", common.LogLevelError, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewModelManager(logger)

	if manager == nil {
		t.Fatal("NewModelManager returned nil")
	}

	if manager.logger == nil {
		t.Error("Expected logger to be set")
	}

	if len(manager.providers) == 0 {
		t.Error("Expected providers to be registered")
	}

	// Test that default providers are registered
	expectedProviders := []string{"openai", "ollama"}
	for _, providerClass := range expectedProviders {
		if _, exists := manager.providers[providerClass]; !exists {
			t.Errorf("Expected provider '%s' to be registered", providerClass)
		}
	}
}

func TestModelManager_RegisterProvider(t *testing.T) {
	logger, err := common.NewLogger("", "", common.LogLevelError, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewModelManager(logger)
	customProvider := &GenericProvider{class: "custom"}

	manager.RegisterProvider("custom", customProvider)

	if _, exists := manager.providers["custom"]; !exists {
		t.Error("Expected custom provider to be registered")
	}
}

func TestModelManager_InitializeClient(t *testing.T) {
	logger, err := common.NewLogger("", "", common.LogLevelError, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewModelManager(logger)

	tests := []struct {
		name      string
		config    ModelConfig
		expectErr bool
	}{
		{
			name: "OpenAI model",
			config: ModelConfig{
				Model:  "gpt-4",
				Class:  "openai",
				APIKey: "test-key",
			},
			expectErr: false,
		},
		{
			name: "Ollama model",
			config: ModelConfig{
				Model: "llama2",
				Class: "ollama",
			},
			expectErr: false,
		},
		{
			name: "OpenAI model missing API key",
			config: ModelConfig{
				Model:  "gpt-4",
				Class:  "openai",
				APIKey: "",
			},
			expectErr: true,
		},
		{
			name: "Unknown model class",
			config: ModelConfig{
				Model:  "custom-model",
				Class:  "unknown",
				APIKey: "test-key",
			},
			expectErr: false,
		},
		{
			name: "Empty class defaults to OpenAI",
			config: ModelConfig{
				Model:  "gpt-3.5-turbo",
				Class:  "",
				APIKey: "test-key",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := manager.InitializeClient(tt.config)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Error("Expected client to be initialized")
				}
			}
		})
	}
}

func TestModelManager_ValidateConfig(t *testing.T) {
	logger, err := common.NewLogger("", "", common.LogLevelError, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewModelManager(logger)

	tests := []struct {
		name      string
		config    ModelConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid OpenAI config",
			config: ModelConfig{
				Model:  "gpt-4",
				Class:  "openai",
				APIKey: "test-key",
			},
			expectErr: false,
		},
		{
			name: "valid Ollama config",
			config: ModelConfig{
				Model: "llama2",
				Class: "ollama",
			},
			expectErr: false,
		},
		{
			name: "OpenAI missing API key",
			config: ModelConfig{
				Model:  "gpt-4",
				Class:  "openai",
				APIKey: "",
			},
			expectErr: true,
			errMsg:    "API key is required for OpenAI models (set API key environment variable or pass via config/flags)",
		},
		{
			name: "OpenAI missing model",
			config: ModelConfig{
				Model:  "",
				Class:  "openai",
				APIKey: "test-key",
			},
			expectErr: true,
			errMsg:    "model name is required for OpenAI models",
		},
		{
			name: "Ollama missing model",
			config: ModelConfig{
				Model: "",
				Class: "ollama",
			},
			expectErr: true,
			errMsg:    "model name is required for Ollama models",
		},
		{
			name: "unknown class valid config",
			config: ModelConfig{
				Model:  "custom-model",
				Class:  "custom",
				APIKey: "test-key",
			},
			expectErr: false,
		},
		{
			name: "unknown class missing model",
			config: ModelConfig{
				Model: "",
				Class: "custom",
			},
			expectErr: true,
			errMsg:    "model name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ValidateConfig(tt.config)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Expected error message %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestOpenAIProvider(t *testing.T) {
	logger, err := common.NewLogger("", "", common.LogLevelError, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	provider := &OpenAIProvider{}

	t.Run("GetProviderName", func(t *testing.T) {
		name := provider.GetProviderName()
		if name != "OpenAI" {
			t.Errorf("Expected provider name 'OpenAI', got '%s'", name)
		}
	})

	t.Run("InitializeClient success", func(t *testing.T) {
		config := ModelConfig{
			Model:  "gpt-4",
			APIKey: "test-key",
		}

		client, err := provider.InitializeClient(config, logger)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if client == nil {
			t.Error("Expected client to be initialized")
		}
	})

	t.Run("InitializeClient missing API key", func(t *testing.T) {
		config := ModelConfig{
			Model:  "gpt-4",
			APIKey: "",
		}

		client, err := provider.InitializeClient(config, logger)
		if err == nil {
			t.Error("Expected error for missing API key")
		}
		if client != nil {
			t.Error("Expected nil client for error case")
		}
	})

	t.Run("ValidateConfig success", func(t *testing.T) {
		config := ModelConfig{
			Model:  "gpt-4",
			APIKey: "test-key",
		}

		err := provider.ValidateConfig(config, logger)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("ValidateConfig missing model", func(t *testing.T) {
		config := ModelConfig{
			Model:  "",
			APIKey: "test-key",
		}

		err := provider.ValidateConfig(config, logger)
		if err == nil {
			t.Error("Expected error for missing model")
		}
	})

	t.Run("ValidateConfig missing API key", func(t *testing.T) {
		config := ModelConfig{
			Model:  "gpt-4",
			APIKey: "",
		}

		err := provider.ValidateConfig(config, logger)
		if err == nil {
			t.Error("Expected error for missing API key")
		}
	})
}

func TestOllamaProvider(t *testing.T) {
	logger, err := common.NewLogger("", "", common.LogLevelError, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	provider := &OllamaProvider{}

	t.Run("GetProviderName", func(t *testing.T) {
		name := provider.GetProviderName()
		if name != "Ollama" {
			t.Errorf("Expected provider name 'Ollama', got '%s'", name)
		}
	})

	t.Run("InitializeClient", func(t *testing.T) {
		config := ModelConfig{
			Model: "llama2",
		}

		client, err := provider.InitializeClient(config, logger)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if client == nil {
			t.Error("Expected client to be initialized")
		}
	})

	t.Run("InitializeClient with custom URL", func(t *testing.T) {
		config := ModelConfig{
			Model:  "llama2",
			APIURL: "http://custom-host:11434/v1",
		}

		client, err := provider.InitializeClient(config, logger)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if client == nil {
			t.Error("Expected client to be initialized")
		}
	})

	t.Run("ValidateConfig success", func(t *testing.T) {
		config := ModelConfig{
			Model: "llama2",
		}

		err := provider.ValidateConfig(config, logger)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("ValidateConfig missing model", func(t *testing.T) {
		config := ModelConfig{
			Model: "",
		}

		err := provider.ValidateConfig(config, logger)
		if err == nil {
			t.Error("Expected error for missing model")
		}
	})
}

func TestGenericProvider(t *testing.T) {
	logger, err := common.NewLogger("", "", common.LogLevelError, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	provider := &GenericProvider{class: "custom"}

	t.Run("GetProviderName", func(t *testing.T) {
		name := provider.GetProviderName()
		expected := "OpenAI-compatible (custom)"
		if name != expected {
			t.Errorf("Expected provider name '%s', got '%s'", expected, name)
		}
	})

	t.Run("InitializeClient with API key", func(t *testing.T) {
		config := ModelConfig{
			Model:  "custom-model",
			APIKey: "test-key",
		}

		client, err := provider.InitializeClient(config, logger)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if client == nil {
			t.Error("Expected client to be initialized")
		}
	})

	t.Run("InitializeClient without API key", func(t *testing.T) {
		config := ModelConfig{
			Model:  "custom-model",
			APIKey: "",
		}

		client, err := provider.InitializeClient(config, logger)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if client == nil {
			t.Error("Expected client to be initialized")
		}
	})

	t.Run("ValidateConfig success", func(t *testing.T) {
		config := ModelConfig{
			Model: "custom-model",
		}

		err := provider.ValidateConfig(config, logger)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("ValidateConfig missing model", func(t *testing.T) {
		config := ModelConfig{
			Model: "",
		}

		err := provider.ValidateConfig(config, logger)
		if err == nil {
			t.Error("Expected error for missing model")
		}
	})
}

func TestConvenienceFunctions(t *testing.T) {
	logger, err := common.NewLogger("", "", common.LogLevelError, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	t.Run("InitializeModelClient", func(t *testing.T) {
		config := ModelConfig{
			Model:  "gpt-4",
			Class:  "openai",
			APIKey: "test-key",
		}

		client, err := InitializeModelClient(config, logger)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if client == nil {
			t.Error("Expected client to be initialized")
		}
	})

	t.Run("ValidateModelConfig", func(t *testing.T) {
		config := ModelConfig{
			Model:  "gpt-4",
			Class:  "openai",
			APIKey: "test-key",
		}

		err := ValidateModelConfig(config, logger)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
