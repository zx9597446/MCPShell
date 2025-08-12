package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/utils"
)

func TestNew(t *testing.T) {
	logger, err := common.NewLogger("", "", common.LogLevelError, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := AgentConfig{
		ToolsFile:  "test.yaml",
		UserPrompt: "test prompt",
		Once:       false,
		Version:    "1.0.0",
		ModelConfig: ModelConfig{
			Model:  "gpt-4",
			APIKey: "test-key",
		},
	}

	agent := New(cfg, logger)

	if agent == nil {
		t.Fatal("New() returned nil")
	}

	if agent.config.ToolsFile != cfg.ToolsFile {
		t.Errorf("Expected ToolsFile %s, got %s", cfg.ToolsFile, agent.config.ToolsFile)
	}

	if agent.config.UserPrompt != cfg.UserPrompt {
		t.Errorf("Expected UserPrompt %s, got %s", cfg.UserPrompt, agent.config.UserPrompt)
	}

	if agent.config.Once != cfg.Once {
		t.Errorf("Expected Once %t, got %t", cfg.Once, agent.config.Once)
	}

	if agent.config.Version != cfg.Version {
		t.Errorf("Expected Version %s, got %s", cfg.Version, agent.config.Version)
	}

	if agent.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestValidate(t *testing.T) {
	logger, err := common.NewLogger("", "", common.LogLevelError, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	tests := []struct {
		name    string
		config  AgentConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: AgentConfig{
				ToolsFile: "test.yaml",
				ModelConfig: ModelConfig{
					Model:  "gpt-4",
					APIKey: "test-key",
				},
			},
			wantErr: false,
		},
		{
			name: "missing tools file",
			config: AgentConfig{
				ToolsFile: "",
				ModelConfig: ModelConfig{
					Model:  "gpt-4",
					APIKey: "test-key",
				},
			},
			wantErr: true,
			errMsg:  "tools configuration file is required",
		},
		{
			name: "missing model",
			config: AgentConfig{
				ToolsFile: "test.yaml",
				ModelConfig: ModelConfig{
					Model:  "",
					APIKey: "test-key",
				},
			},
			wantErr: true,
			errMsg:  "LLM model is required",
		},
		{
			name: "missing API key",
			config: AgentConfig{
				ToolsFile: "test.yaml",
				ModelConfig: ModelConfig{
					Model:  "gpt-4",
					APIKey: "",
				},
			},
			wantErr: true,
			errMsg:  "API key is required (set API key environment variable or pass via config/flags)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := New(tt.config, logger)
			err := agent.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
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

func TestSetupConversation(t *testing.T) {
	logger, err := common.NewLogger("", "", common.LogLevelError, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	tests := []struct {
		name           string
		config         AgentConfig
		expectedLength int
		hasUserPrompt  bool
	}{
		{
			name: "with user prompt",
			config: AgentConfig{
				UserPrompt: "test user prompt",
				ModelConfig: ModelConfig{
					Prompts: common.PromptsConfig{
						System: []string{"test system prompt"},
					},
				},
			},
			expectedLength: 2,
			hasUserPrompt:  true,
		},
		{
			name: "without user prompt",
			config: AgentConfig{
				UserPrompt: "",
				ModelConfig: ModelConfig{
					Prompts: common.PromptsConfig{
						System: []string{"test system prompt"},
					},
				},
			},
			expectedLength: 1,
			hasUserPrompt:  false,
		},
		{
			name: "no system prompt - uses default",
			config: AgentConfig{
				UserPrompt: "test user prompt",
				ModelConfig: ModelConfig{
					Prompts: common.PromptsConfig{},
				},
			},
			expectedLength: 2,
			hasUserPrompt:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := New(tt.config, logger)
			messages := agent.setupConversation()

			if len(messages) != tt.expectedLength {
				t.Errorf("Expected %d messages, got %d", tt.expectedLength, len(messages))
			}

			// First message should always be system message
			if len(messages) > 0 && messages[0].Role != openai.ChatMessageRoleSystem {
				t.Errorf("Expected first message to be system message, got %s", messages[0].Role)
			}

			// Check if user message is present when expected
			if tt.hasUserPrompt {
				if len(messages) < 2 {
					t.Error("Expected user message but didn't find it")
				} else if messages[1].Role != openai.ChatMessageRoleUser {
					t.Errorf("Expected second message to be user message, got %s", messages[1].Role)
				}
			}

			// System prompt should always contain termination instruction
			if len(messages) > 0 && !strings.Contains(messages[0].Content, "TERMINATE") {
				t.Error("Expected system prompt to contain termination instruction")
			}
		})
	}
}

func TestInitializeOpenAIClient(t *testing.T) {
	logger, err := common.NewLogger("", "", common.LogLevelError, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	tests := []struct {
		name   string
		config AgentConfig
	}{
		{
			name: "basic client initialization",
			config: AgentConfig{
				ModelConfig: ModelConfig{
					Model:  "gpt-4",
					APIKey: "test-key",
				},
			},
		},
		{
			name: "client with custom API URL",
			config: AgentConfig{
				ModelConfig: ModelConfig{
					Model:  "gpt-4",
					APIKey: "test-key",
					APIURL: "https://custom.api.url",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := New(tt.config, logger)
			client := agent.initializeOpenAIClient()

			if client == nil {
				t.Error("Expected OpenAI client to be initialized")
			}
		})
	}
}

// TestAgentWithOllama tests the agent with a real Ollama model
// This test requires Ollama to be running and a tool-capable model to be available
func TestAgentWithOllama(t *testing.T) {
	// Use the test utilities to check if Ollama is running and get a tool-capable model
	modelName := utils.RequireOllamaWithTools(t)

	t.Logf("Running integration test with model: %s", modelName)

	// Create a logger for the test
	logger, err := common.NewLogger("", "", common.LogLevelInfo, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create a temporary test config file
	testConfig := createTestConfigFile(t)
	defer func() {
		if err := os.Remove(testConfig); err != nil && !os.IsNotExist(err) {
			t.Logf("failed to remove test config: %v", err)
		}
	}()

	// Create agent configuration for Ollama
	cfg := AgentConfig{
		ToolsFile:  testConfig,
		UserPrompt: "What is the current date? Use the date tool to find out.",
		Once:       true,
		Version:    "test",
		ModelConfig: ModelConfig{
			Model:  modelName,
			APIURL: "http://localhost:11434/v1", // Ollama's OpenAI-compatible endpoint
			APIKey: "ollama",                    // Ollama doesn't require a real API key
			Prompts: common.PromptsConfig{
				System: []string{"You are a helpful assistant that can use tools to answer questions."},
			},
		},
	}

	// Create and validate the agent
	agent := New(cfg, logger)
	if agent == nil {
		t.Fatal("Failed to create agent")
	}

	err = agent.Validate()
	if err != nil {
		t.Fatalf("Agent validation failed: %v", err)
	}

	// Test that the model supports tools
	if !utils.IsModelToolCapable(modelName) {
		t.Errorf("Model %s should be tool-capable according to our test utilities", modelName)
	}

	// Test OpenAI client initialization
	client := agent.initializeOpenAIClient()
	if client == nil {
		t.Fatal("Failed to initialize OpenAI client")
	}

	// Test conversation setup
	messages := agent.setupConversation()
	if len(messages) < 2 {
		t.Fatal("Expected at least 2 messages (system + user)")
	}

	if messages[0].Role != "system" {
		t.Error("First message should be system message")
	}

	if messages[1].Role != "user" {
		t.Error("Second message should be user message")
	}

	if !strings.Contains(messages[1].Content, "date") {
		t.Error("User message should contain 'date' from our test prompt")
	}

	// Test that we can call the model (basic connectivity test)
	// We'll do a simple test call without tools to verify the connection works
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a simple test request
	testMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are a helpful assistant. Respond briefly.",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "Say 'Hello, MCPShell test!' and nothing else.",
		},
	}

	req := openai.ChatCompletionRequest{
		Model:       cfg.Model,
		Messages:    testMessages,
		MaxTokens:   50,
		Temperature: 0.1,
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create chat completion: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("No response choices returned")
	}

	responseText := resp.Choices[0].Message.Content
	if responseText == "" {
		t.Fatal("Empty response from model")
	}

	t.Logf("Model response: %s", responseText)

	// Verify the response contains expected content
	if !strings.Contains(strings.ToLower(responseText), "hello") {
		t.Errorf("Expected response to contain 'hello', got: %s", responseText)
	}

	t.Log("Ollama integration test completed successfully")
}

// createTestConfigFile creates a temporary configuration file for testing
func createTestConfigFile(t *testing.T) string {
	testConfig := `
tools:
  - name: "date"
    description: "Get the current date and time"
    runner: "exec"
    command: "date"
    parameters: []
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	err := os.WriteFile(configFile, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	return configFile
}
