package root

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/inercia/MCPShell/pkg/agent"
)

// buildAgentConfig creates an AgentConfig by merging command-line flags with configuration file
func buildAgentConfig() (agent.AgentConfig, error) {
	// Load configuration from file
	config, err := agent.GetConfig()
	if err != nil {
		return agent.AgentConfig{}, fmt.Errorf("failed to load config: %w", err)
	}

	// Start with default model from config file
	var modelConfig agent.ModelConfig
	if defaultModel := config.GetDefaultModel(); defaultModel != nil {
		modelConfig = *defaultModel
	}

	// Override with command-line flags if provided
	if agentModel != "" {
		// Check if the specified model exists in config
		if configModel := config.GetModelByName(agentModel); configModel != nil {
			modelConfig = *configModel
		} else {
			// Use command-line model name if not found in config
			modelConfig.Model = agentModel
		}
	}

	// Merge system prompts from config file and command-line
	if agentSystemPrompt != "" {
		// Join system prompts from config with command-line system prompt
		var allSystemPrompts []string

		// Add existing system prompts from config
		if modelConfig.Prompts.HasSystemPrompts() {
			allSystemPrompts = append(allSystemPrompts, modelConfig.Prompts.System...)
		}

		// Add command-line system prompt
		allSystemPrompts = append(allSystemPrompts, agentSystemPrompt)

		// Update the prompts config with merged system prompts
		modelConfig.Prompts.System = allSystemPrompts
		// Clear user prompts as they should be ignored from config
		modelConfig.Prompts.User = nil
	} else {
		// No command-line system prompt provided, but still clear user prompts from config
		modelConfig.Prompts.User = nil
	}
	if agentOpenAIApiKey != "" {
		modelConfig.APIKey = agentOpenAIApiKey
	}
	if agentOpenAIApiURL != "" {
		modelConfig.APIURL = agentOpenAIApiURL
	}

	// If no API key is set, try environment variable or handle template value
	switch modelConfig.APIKey {
	case "":
		modelConfig.APIKey = os.Getenv("OPENAI_API_KEY")
	case "${OPENAI_API_KEY}":
		// Handle environment variable substitution
		modelConfig.APIKey = os.Getenv("OPENAI_API_KEY")
	}

	return agent.AgentConfig{
		ToolsFile:   toolsFile,
		UserPrompt:  agentUserPrompt,
		Once:        agentOnce,
		Version:     version,
		ModelConfig: modelConfig,
	}, nil
}

// agentCommand is a command that executes the MCPShell as an agent
var agentCommand = &cobra.Command{
	Use:   "agent",
	Short: "Execute the MCPShell as an agent",
	Long: `

The agent command will execute the MCPShell as an agent, connecting to a remote LLM.

Configuration is loaded from ~/.mcpshell/agent.yaml and can be overridden with command-line flags.
The configuration file should contain model definitions with their API keys and prompts.

For example, you can do:

$ mcpshell agent --tools=examples/config.yaml \
     --model "gpt-4o" \
     --system-prompt "You are a helpful assistant that debugs performance issues" \
     --user-prompt "I am having trouble with my computer. It is slow and I think it is due to the CPU usage."

If a model is configured as default in the agent configuration file, you can omit the --model flag:

$ mcpshell agent --tools=examples/config.yaml \
     --user-prompt "I am having trouble with my computer. It is slow and I think it is due to the CPU usage."

You can also provide the initial user prompt as positional arguments:

$ mcpshell agent I am having trouble with my computer. It is slow and I think it is due to the CPU usage.

The agent will try to debug the issue with the given tools.
`,
	Args: cobra.ArbitraryArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// If --user-prompt is not provided but positional args exist, join them as the user prompt
		if agentUserPrompt == "" && len(args) > 0 {
			agentUserPrompt = strings.Join(args, " ")
		}

		// Initialize logger
		logger, err := initLogger()
		if err != nil {
			return err
		}

		// Build agent configuration
		agentConfig, err := buildAgentConfig()
		if err != nil {
			return err
		}

		// Validate agent configuration
		agentInstance := agent.New(agentConfig, logger)
		if err := agentInstance.Validate(); err != nil {
			return err
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// If --user-prompt is not provided but positional args exist, join them as the user prompt
		if agentUserPrompt == "" && len(args) > 0 {
			agentUserPrompt = strings.Join(args, " ")
		}

		// Initialize logger
		logger, err := initLogger()
		if err != nil {
			return err
		}

		// Build agent configuration
		agentConfig, err := buildAgentConfig()
		if err != nil {
			return err
		}

		// Create agent instance
		agentInstance := agent.New(agentConfig, logger)

		// Create channels for user input and agent output
		userInput := make(chan string)
		agentOutput := make(chan string)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Setup signal handling for graceful shutdown
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-signalChan:
				logger.Info("Received interrupt signal, shutting down...")
				cancel()
			case <-ctx.Done():
			}
		}()

		// Start a goroutine to read user input only when not in --once mode
		if !agentConfig.Once {
			wg.Add(1)
			go func() {
				defer wg.Done()
				scanner := bufio.NewScanner(os.Stdin)
				for {
					select {
					case <-ctx.Done():
						return
					default:
						if scanner.Scan() {
							userInput <- scanner.Text()
						} else {
							close(userInput)
							return
						}
					}
				}
			}()
		}

		// Start the agent
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := agentInstance.Run(ctx, userInput, agentOutput); err != nil {
				logger.Error("Agent encountered an error: %v", err)
			}
		}()

		// Print agent output
		for output := range agentOutput {
			fmt.Println(output)
		}

		wg.Wait()
		return nil
	},
}

// init adds the agent command to the root command
func init() {
	// Add agent command to root
	rootCmd.AddCommand(agentCommand)

	// Add agent-specific flags
	agentCommand.Flags().StringVarP(&agentModel, "model", "m", "", "LLM model to use (required)")
	agentCommand.Flags().StringVarP(&agentSystemPrompt, "system-prompt", "s", "You are a helpful assistant.", "System prompt for the LLM")
	agentCommand.Flags().StringVarP(&agentUserPrompt, "user-prompt", "u", "", "Initial user prompt for the LLM")
	agentCommand.Flags().StringVarP(&agentOpenAIApiKey, "openai-api-key", "k", "", "OpenAI API key (or set OPENAI_API_KEY environment variable)")
	agentCommand.Flags().StringVarP(&agentOpenAIApiURL, "openai-api-url", "b", "", "Base URL for the OpenAI API (optional)")
	agentCommand.Flags().BoolVarP(&agentOnce, "once", "o", false, "Exit after receiving a final response from the LLM (one-shot mode)")

	// Add config subcommand
	agentCommand.AddCommand(agentConfigCommand)
}
