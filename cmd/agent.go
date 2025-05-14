package root

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/config"
	"github.com/inercia/MCPShell/pkg/server"
)

// Command-line flags for agent
var (
	agentConfigFile   string
	agentLogFile      string
	agentLogLevel     string
	agentModel        string
	agentSystemPrompt string
	agentUserPrompt   string
	agentOpenAIApiKey string
	agentOpenAIApiURL string
	agentOnce         bool
)

// agentCommand is a command that executes the MCPShell as an agent
var agentCommand = &cobra.Command{
	Use:   "agent",
	Short: "Execute the MCPShell as an agent",
	Long: `

The agent command will execute the MCPShell as an agent, connecting to a remote LLM.
For example, you can do

$ mcpshell agent --configfile=examples/config.yaml \
     --model "gpt-4o" \
     --system-prompt "You are a helpful assistant that debugs performance issues" \
	 --user-prompt "I am having trouble with my computer. It is slow and I think it is due to the CPU usage."

and the agent will try to debug the issue with the given tools.
`,
	Args: cobra.NoArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger
		level := common.LogLevelFromString(agentLogLevel)
		logger, err := common.NewLogger("[mcpshell] ", agentLogFile, level, true)
		if err != nil {
			return fmt.Errorf("failed to set up logger: %w", err)
		}

		// Set global logger for application-wide use
		common.SetLogger(logger)

		// Setup panic handler
		defer common.RecoverPanic()

		logger.Info("Starting MCPShell agent")

		// Check if config file is provided
		if agentConfigFile == "" {
			logger.Error("Configuration file is required")
			return fmt.Errorf("configuration file is required. Use --config or -c flag to specify the path")
		}

		// Check if model is provided
		if agentModel == "" {
			logger.Error("LLM model is required")
			return fmt.Errorf("LLM model is required. Use --model flag to specify the model")
		}

		// Check if API key is provided or in environment
		if agentOpenAIApiKey == "" {
			agentOpenAIApiKey = os.Getenv("OPENAI_API_KEY")
			if agentOpenAIApiKey == "" {
				logger.Error("OpenAI API key is required")
				return fmt.Errorf("OpenAI API key is required. Use --api-key flag or set OPENAI_API_KEY environment variable")
			}
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the logger
		logger := common.GetLogger()

		// Setup panic handler
		defer common.RecoverPanic()

		// Load the configuration file (local or remote)
		localConfigPath, cleanup, err := config.ResolveConfigPath(agentConfigFile, logger)
		if err != nil {
			logger.Error("Failed to load configuration: %v", err)
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Ensure temporary files are cleaned up
		defer cleanup()

		// Load configuration to access prompts
		cfg, err := config.NewConfigFromFile(localConfigPath)
		if err != nil {
			logger.Error("Failed to parse configuration: %v", err)
			return fmt.Errorf("failed to parse configuration: %w", err)
		}

		// Initialize MCP server to get tools
		logger.Info("Initializing MCP server")
		srv := server.New(server.Config{
			ConfigFile: localConfigPath,
			Logger:     logger,
			Version:    version,
		})

		// Create the server instance (but don't start it)
		if err := srv.CreateServer(); err != nil {
			logger.Error("Failed to create MCP server: %v", err)
			return fmt.Errorf("failed to create MCP server: %w", err)
		}

		// Initialize OpenAI client
		openaiConfig := openai.DefaultConfig(agentOpenAIApiKey)
		if agentOpenAIApiURL != "" {
			openaiConfig.BaseURL = agentOpenAIApiURL
		}
		client := openai.NewClientWithConfig(openaiConfig)
		logger.Info("Initialized OpenAI client with model: %s", agentModel)

		// Convert MCP tools to OpenAI tools
		openaiTools, err := srv.GetOpenAITools()
		if err != nil {
			logger.Error("Failed to convert MCP tools to OpenAI format: %v", err)
			return fmt.Errorf("failed to convert MCP tools to OpenAI format: %w", err)
		}
		logger.Info("Retrieved %d tools from MCP server", len(openaiTools))

		// Add termination instructions to the system prompt
		systemPrompt := agentSystemPrompt
		if systemPrompt == "" && len(cfg.Prompts) > 0 {
			// Use system prompts from config if available
			var systemPrompts []string
			for _, prompt := range cfg.Prompts {
				systemPrompts = append(systemPrompts, prompt.System...)
			}
			if len(systemPrompts) > 0 {
				systemPrompt = strings.Join(systemPrompts, "\n\n")
				logger.Info("Using system prompt from config file")
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
		if agentUserPrompt != "" {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: agentUserPrompt,
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
						logger.Info("Added user prompt from config file")
					}
				}
			}
		}

		// Main interaction loop
		scanner := bufio.NewScanner(os.Stdin)
		ctx := context.Background()

		for {
			// Create the chat completion request
			req := openai.ChatCompletionRequest{
				Model:    agentModel,
				Messages: messages,
				Tools:    openaiTools,
			}

			// Get response from the model
			resp, err := client.CreateChatCompletion(ctx, req)
			if err != nil {
				logger.Error("Error getting LLM response: %v", err)
				return fmt.Errorf("error getting LLM response: %w", err)
			}

			// Check for tool calls
			if len(resp.Choices) > 0 && len(resp.Choices[0].Message.ToolCalls) > 0 {
				// Process tool calls
				assistantMsg := resp.Choices[0].Message

				// Add the assistant message with tool calls
				messages = append(messages, assistantMsg)

				// Print the assistant message if it has content
				if assistantMsg.Content != "" {
					fmt.Printf("Assistant: %s\n", assistantMsg.Content)
				}

				// Process each tool call
				for _, call := range assistantMsg.ToolCalls {
					logger.Info("Processing tool call: %s", call.Function.Name)

					// Log the raw arguments for debugging
					logger.Debug("Raw tool arguments: %s", call.Function.Arguments)

					// Parse the arguments
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
						logger.Error("Failed to parse tool arguments: %v", err)
						toolResultMsg := openai.ChatCompletionMessage{
							Role:       openai.ChatMessageRoleTool,
							Content:    fmt.Sprintf("Error: Failed to parse arguments - %v", err),
							ToolCallID: call.ID,
						}
						messages = append(messages, toolResultMsg)
						fmt.Printf("Tool %s error: Failed to parse arguments\n", call.Function.Name)
						continue
					}

					// Log the parsed arguments
					argsJSON, _ := json.MarshalIndent(args, "", "  ")
					logger.Debug("Parsed tool arguments: %s", string(argsJSON))

					// Verify required arguments are present and not empty
					if args["filename"] == nil || args["filename"] == "" {
						logger.Error("Missing or empty 'filename' argument")
						toolResultMsg := openai.ChatCompletionMessage{
							Role:       openai.ChatMessageRoleTool,
							Content:    "Error: Missing or empty 'filename' argument",
							ToolCallID: call.ID,
						}
						messages = append(messages, toolResultMsg)
						fmt.Printf("Tool %s error: Missing or empty 'filename' argument\n", call.Function.Name)
						continue
					}

					// Ensure string values for all arguments
					for key, value := range args {
						if strValue, ok := value.(string); ok {
							if strValue == "" {
								logger.Info("Empty string value for argument: %s", key)
							}
						} else {
							// Try to convert to string if needed
							args[key] = fmt.Sprintf("%v", value)
							logger.Info("Converted non-string value for argument: %s = %v -> %s", key, value, args[key])
						}
					}

					// Execute the tool
					toolResult, err := srv.ExecuteTool(ctx, call.Function.Name, args)
					if err != nil {
						logger.Error("Failed to execute tool '%s': %v", call.Function.Name, err)
						toolResultMsg := openai.ChatCompletionMessage{
							Role:       openai.ChatMessageRoleTool,
							Content:    fmt.Sprintf("Error: %v", err),
							ToolCallID: call.ID,
						}
						messages = append(messages, toolResultMsg)
						fmt.Printf("Tool %s error: %v\n", call.Function.Name, err)
						continue
					}

					// Add the result
					toolResultMsg := openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    toolResult,
						ToolCallID: call.ID,
					}
					messages = append(messages, toolResultMsg)
					fmt.Printf("Tool %s result: %s\n", call.Function.Name, toolResult)
				}
			} else if len(resp.Choices) > 0 {
				// No tool calls, just print the message
				assistantMessage := resp.Choices[0].Message
				messages = append(messages, assistantMessage)

				// Print the assistant response
				fmt.Printf("Assistant: %s\n", assistantMessage.Content)

				// Check for termination
				if strings.Contains(strings.ToUpper(assistantMessage.Content), "TERMINATE") {
					logger.Info("LLM requested termination, ending the conversation")
					fmt.Println("Conversation terminated.")
					return nil
				}

				// Exit if in one-shot mode
				if agentOnce {
					logger.Info("One-shot mode enabled, ending conversation after receiving response")
					return nil
				}

				// Get user input
				fmt.Print("\nYou: ")
				if !scanner.Scan() {
					break
				}

				userInput := scanner.Text()
				messages = append(messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: userInput,
				})
			}
		}

		return nil
	},
}

// init adds the agent command to the root command
func init() {
	// Add agent command to root
	rootCmd.AddCommand(agentCommand)

	// Add flags
	agentCommand.Flags().StringVarP(&agentConfigFile, "config", "c", "", "Path to the YAML configuration file or URL (required)")
	agentCommand.Flags().StringVarP(&agentLogFile, "logfile", "l", "", "Path to the log file (optional)")
	agentCommand.Flags().StringVarP(&agentLogLevel, "log-level", "", "info", "Log level: none, error, info, debug")
	agentCommand.Flags().StringVarP(&agentModel, "model", "m", "", "LLM model to use (required)")
	agentCommand.Flags().StringVarP(&agentSystemPrompt, "system-prompt", "s", "You are a helpful assistant.", "System prompt for the LLM")
	agentCommand.Flags().StringVarP(&agentUserPrompt, "user-prompt", "u", "", "Initial user prompt for the LLM")
	agentCommand.Flags().StringVarP(&agentOpenAIApiKey, "openai-api-key", "k", "", "OpenAI API key (or set OPENAI_API_KEY environment variable)")
	agentCommand.Flags().StringVarP(&agentOpenAIApiURL, "openai-api-url", "b", "", "Base URL for the OpenAI API (optional)")
	agentCommand.Flags().BoolVarP(&agentOnce, "once", "o", false, "Exit after receiving a final response from the LLM (one-shot mode)")

	// Mark required flags
	_ = agentCommand.MarkFlagRequired("config")
	_ = agentCommand.MarkFlagRequired("model")
}
