package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/config"
	"github.com/inercia/MCPShell/pkg/server"
)

// AgentConfig holds the configuration for the agent
type AgentConfig struct {
	ConfigFile   string
	Model        string
	SystemPrompt string
	UserPrompt   string
	OpenAIApiKey string
	OpenAIApiURL string
	Once         bool
	Version      string
}

// Agent represents an MCP agent
type Agent struct {
	config AgentConfig
	logger *common.Logger
}

// New creates a new agent instance
func New(cfg AgentConfig, logger *common.Logger) *Agent {
	return &Agent{
		config: cfg,
		logger: logger,
	}
}

// Validate checks if the configuration is valid
func (a *Agent) Validate() error {
	// Check if config file is provided
	if a.config.ConfigFile == "" {
		a.logger.Error("Configuration file is required")
		return fmt.Errorf("configuration file is required")
	}

	// Check if model is provided
	if a.config.Model == "" {
		a.logger.Error("LLM model is required")
		return fmt.Errorf("LLM model is required")
	}

	// Check if API key is provided or in environment
	if a.config.OpenAIApiKey == "" {
		// Try to get from environment if not provided in config (e.g. flags)
		// Note: This logic might be better placed in the CLI or a dedicated config loader
		// if agentOpenAIApiKey := os.Getenv("OPENAI_API_KEY"); agentOpenAIApiKey != "" {
		//  a.config.OpenAIApiKey = agentOpenAIApiKey
		// } else {
		a.logger.Error("OpenAI API key is required")
		return fmt.Errorf("OpenAI API key is required (set OPENAI_API_KEY environment variable or pass via config/flags)")
		// }
	}

	return nil
}

// Run executes the agent
func (a *Agent) Run(ctx context.Context, userInput chan string, agentOutput chan string) error {
	// Setup panic handler
	defer common.RecoverPanic()
	defer close(agentOutput) // Ensure agentOutput is closed when Run exits

	// Load the configuration file (local or remote)
	localConfigPath, cleanup, err := config.ResolveConfigPath(a.config.ConfigFile, a.logger)
	if err != nil {
		a.logger.Error("Failed to load configuration: %v", err)
		agentOutput <- fmt.Sprintf("Error: Failed to load configuration: %v", err)
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Ensure temporary files are cleaned up
	defer cleanup()

	// Load configuration to access prompts
	cfg, err := config.NewConfigFromFile(localConfigPath)
	if err != nil {
		a.logger.Error("Failed to parse configuration: %v", err)
		agentOutput <- fmt.Sprintf("Error: Failed to parse configuration: %v", err)
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Initialize MCP server to get tools
	a.logger.Info("Initializing MCP server")
	srv := server.New(server.Config{
		ConfigFile: localConfigPath,
		Logger:     a.logger,
		Version:    a.config.Version,
	})

	// Create the server instance (but don't start it)
	if err := srv.CreateServer(); err != nil {
		a.logger.Error("Failed to create MCP server: %v", err)
		agentOutput <- fmt.Sprintf("Error: Failed to create MCP server: %v", err)
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	// Initialize OpenAI client
	openaiConfig := openai.DefaultConfig(a.config.OpenAIApiKey)
	if a.config.OpenAIApiURL != "" {
		openaiConfig.BaseURL = a.config.OpenAIApiURL
	}
	client := openai.NewClientWithConfig(openaiConfig)
	a.logger.Info("Initialized OpenAI client with model: %s", a.config.Model)

	// Convert MCP tools to OpenAI tools
	openaiTools, err := srv.GetOpenAITools()
	if err != nil {
		a.logger.Error("Failed to convert MCP tools to OpenAI format: %v", err)
		agentOutput <- fmt.Sprintf("Error: Failed to convert MCP tools to OpenAI format: %v", err)
		return fmt.Errorf("failed to convert MCP tools to OpenAI format: %w", err)
	}
	a.logger.Info("Retrieved %d tools from MCP server", len(openaiTools))

	// Add termination instructions to the system prompt
	systemPrompt := a.config.SystemPrompt
	if systemPrompt == "" && len(cfg.Prompts) > 0 {
		// Use system prompts from config if available
		var systemPrompts []string
		for _, prompt := range cfg.Prompts {
			systemPrompts = append(systemPrompts, prompt.System...)
		}
		if len(systemPrompts) > 0 {
			systemPrompt = strings.Join(systemPrompts, "\n\n")
			a.logger.Info("Using system prompt from config file")
		}
	}

	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant."
	}

	if !strings.Contains(systemPrompt, "terminate the conversation") {
		systemPrompt += "\n\nWhen you have completed your task, please type 'TERMINATE' to end the conversation."
	}

	// Start the conversation
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	// Add user prompt if provided or from config
	if a.config.UserPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: a.config.UserPrompt,
		})
	} else if len(cfg.Prompts) > 0 {
		// Add user prompts from config
		for _, promptSet := range cfg.Prompts {
			for _, userPrompt := range promptSet.User {
				if userPrompt != "" {
					messages = append(messages, openai.ChatCompletionMessage{
						Role:    openai.ChatMessageRoleUser,
						Content: userPrompt,
					})
					a.logger.Info("Added user prompt from config file")
				}
			}
		}
	}

	// Create a single-run context if in --once mode
	var singleRunCtx context.Context
	var singleRunCancel context.CancelFunc

	if a.config.Once {
		// Create a context with a timeout to ensure we don't get stuck in --once mode
		singleRunCtx, singleRunCancel = context.WithTimeout(ctx, 30*time.Second)
		defer singleRunCancel()

		// Replace the main context with our single-run context for the duration of this Run
		ctx = singleRunCtx

		a.logger.Info("Running in one-shot mode with 30s safety timeout")
	}

	// Main interaction loop
	for {
		// Create the chat completion request
		req := openai.ChatCompletionRequest{
			Model:    a.config.Model,
			Messages: messages,
			Tools:    openaiTools,
		}

		// Get response from the model
		var resp openai.ChatCompletionResponse
		var llmErr error

		// Perform LLM call in a separate goroutine to allow context cancellation
		done := make(chan struct{})
		go func() {
			defer close(done)
			resp, llmErr = client.CreateChatCompletion(ctx, req)
		}()

		select {
		case <-ctx.Done():
			a.logger.Info("Context cancelled, terminating agent Run loop.")
			return ctx.Err()
		case <-done:
			if llmErr != nil {
				a.logger.Error("Error getting LLM response: %v", llmErr)
				agentOutput <- fmt.Sprintf("Error: Error getting LLM response: %v", llmErr)
				return fmt.Errorf("error getting LLM response: %w", llmErr)
			}
		}

		// Process the response - first, check if we have any choices
		if len(resp.Choices) == 0 {
			a.logger.Error("No choices received from LLM")
			agentOutput <- "Error: No response from LLM."
			return fmt.Errorf("no choices received from LLM")
		}

		// Get the response message
		respMsg := resp.Choices[0].Message
		messages = append(messages, respMsg)

		// Check for tool calls
		if len(respMsg.ToolCalls) > 0 {
			// Process tool calls
			if respMsg.Content != "" {
				agentOutput <- fmt.Sprintf("Assistant: %s", respMsg.Content)
			}

			// Process each tool call
			for _, call := range respMsg.ToolCalls {
				a.logger.Info("Processing tool call: %s", call.Function.Name)
				a.logger.Debug("Raw tool arguments: %s", call.Function.Arguments)

				// Parse the arguments
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
					a.logger.Error("Failed to parse tool arguments: %v", err)
					toolResultContent := fmt.Sprintf("Error: Failed to parse arguments - %v", err)
					toolResultMsg := openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    toolResultContent,
						ToolCallID: call.ID,
					}
					messages = append(messages, toolResultMsg)
					agentOutput <- fmt.Sprintf("Tool %s error: Failed to parse arguments", call.Function.Name)
					continue
				}

				// Log the parsed arguments
				argsJSON, _ := json.MarshalIndent(args, "", "  ")
				a.logger.Debug("Parsed tool arguments: %s", string(argsJSON))

				// Convert non-string arguments to strings
				for key, value := range args {
					if _, ok := value.(string); !ok && value != nil {
						args[key] = fmt.Sprintf("%v", value)
					}
				}

				// Execute the tool
				toolResult, err := srv.ExecuteTool(ctx, call.Function.Name, args)
				if err != nil {
					a.logger.Error("Failed to execute tool '%s': %v", call.Function.Name, err)
					errorContent := fmt.Sprintf("Error: %v", err)
					toolResultMsg := openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    errorContent,
						ToolCallID: call.ID,
					}
					messages = append(messages, toolResultMsg)
					agentOutput <- fmt.Sprintf("Tool %s error: %v", call.Function.Name, err)
					continue
				}

				// Add the result
				toolResultMsg := openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    toolResult,
					ToolCallID: call.ID,
				}
				messages = append(messages, toolResultMsg)
				agentOutput <- fmt.Sprintf("Tool %s result: %s", call.Function.Name, toolResult)
			}
		} else {
			// No tool calls, just print the message
			agentOutput <- fmt.Sprintf("Assistant: %s", respMsg.Content)

			// Check for termination request from LLM
			if strings.Contains(strings.ToUpper(respMsg.Content), "TERMINATE") {
				a.logger.Info("LLM requested termination, ending the conversation")
				agentOutput <- "Conversation terminated."
				return nil
			}
		}

		// Exit after first response if in one-shot mode
		if a.config.Once {
			a.logger.Info("One-shot mode enabled, ending after first interaction")
			agentOutput <- "One-shot mode: Completed."
			
			// Since we skipped creating the stdin reader goroutine in cmd/agent.go,
			// we need to close the userInput channel here in one-shot mode
			close(userInput)
			
			return nil
		}

		// Get user input for the next interaction (skipped in one-shot mode)
		agentOutput <- "\nYou: "
		select {
		case <-ctx.Done():
			a.logger.Info("Context cancelled, terminating agent Run loop.")
			return ctx.Err()
		case input, ok := <-userInput:
			if !ok {
				a.logger.Info("User input channel closed, terminating conversation.")
				agentOutput <- "User input closed. Conversation terminated."
				return nil
			}
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: input,
			})
		}
	}
}
