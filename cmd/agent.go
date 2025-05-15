package root

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/inercia/MCPShell/pkg/agent"
	"github.com/inercia/MCPShell/pkg/common"
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
		logger := common.GetLogger()

		agentConfig := agent.AgentConfig{
			ConfigFile:   agentConfigFile,
			Model:        agentModel,
			SystemPrompt: agentSystemPrompt,
			UserPrompt:   agentUserPrompt,
			OpenAIApiKey: agentOpenAIApiKey,
			OpenAIApiURL: agentOpenAIApiURL,
			Once:         agentOnce,
			Version:      version,
		}

		a := agent.New(agentConfig, logger)

		if err := a.Validate(); err != nil {
			return fmt.Errorf("agent validation failed: %w", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle Ctrl+C (SIGINT) and SIGTERM to gracefully shut down
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			logger.Info("Received interrupt signal, cancelling agent context...")
			cancel()
		}()

		userInputChan := make(chan string)
		agentOutputChan := make(chan string)

		var wg sync.WaitGroup

		// Goroutine to read from stdin and send to userInputChan
		if !agentOnce {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer close(userInputChan)
				scanner := bufio.NewScanner(os.Stdin)
				for scanner.Scan() {
					select {
					case userInputChan <- scanner.Text():
					case <-ctx.Done():
						logger.Info("Context cancelled, stopping stdin reader.")
						return
					}
				}
				if err := scanner.Err(); err != nil {
					logger.Error("Error reading from stdin: %v", err)
				}
				logger.Info("Stdin scanner finished.")
			}()
		} else {
			// In one-shot mode, we'll close the channel when Run completes
			logger.Info("One-shot mode, skipping stdin reader.")
		}

		// Goroutine to read from agentOutputChan and print to stdout
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case output, ok := <-agentOutputChan:
					if !ok {
						logger.Info("Agent output channel closed, stdout writer finishing.")
						return
					}
					fmt.Print(output)
				case <-ctx.Done():
					logger.Info("Context cancelled, stopping stdout writer.")
					for output := range agentOutputChan {
						fmt.Print(output)
					}
					return
				}
			}
		}()

		err := a.Run(ctx, userInputChan, agentOutputChan)

		cancel()

		logger.Info("Waiting for I/O goroutines to finish...")
		wg.Wait()
		logger.Info("All goroutines finished.")

		if err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
				logger.Info("Agent run was cancelled: %v", err)
				return nil
			}
			logger.Error("Agent execution failed: %v", err)
			return fmt.Errorf("agent execution failed: %w", err)
		}

		logger.Info("Agent finished successfully.")
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
