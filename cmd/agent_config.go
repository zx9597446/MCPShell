package root

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/inercia/MCPShell/pkg/agent"
	"github.com/inercia/MCPShell/pkg/utils"
)

// agentConfigCommand is the parent command for agent configuration subcommands
var agentConfigCommand = &cobra.Command{
	Use:   "config",
	Short: "Manage agent configuration",
	Long: `

The config command provides subcommands to manage agent configuration files.

Available subcommands:
- create: Create a default agent configuration file
- show: Display the current agent configuration
`,
}

// agentConfigCreateCommand creates a default agent configuration file
var agentConfigCreateCommand = &cobra.Command{
	Use:   "create",
	Short: "Create a default agent configuration file",
	Long: `

Creates a default agent configuration file at ~/.mcpshell/agent.yaml.

If the file already exists, it will be overwritten with the default configuration.
The default configuration includes sample models and prompts that you can customize.

Example:
$ mcpshell agent config create
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, err := initLogger()
		if err != nil {
			return err
		}

		// Create the default config file
		if err := agent.CreateDefaultConfigForce(); err != nil {
			logger.Error("Failed to create default config: %v", err)
			return fmt.Errorf("failed to create default config: %w", err)
		}

		mcpShellHome, err := utils.GetMCPShellHome()
		if err != nil {
			return fmt.Errorf("failed to get MCPShell home directory: %w", err)
		}

		configPath := filepath.Join(mcpShellHome, "agent.yaml")
		fmt.Printf("Default agent configuration created at: %s\n", configPath)
		fmt.Println("You can now edit this file to customize your agent settings.")

		return nil
	},
}

// agentConfigShowCommand displays the current agent configuration
var agentConfigShowCommand = &cobra.Command{
	Use:   "show",
	Short: "Display the current agent configuration",
	Long: `

Displays the current agent configuration in a pretty-printed format.

The configuration is loaded from ~/.mcpshell/agent.yaml and parsed to show
the available models, their settings, and which model is set as default.

Example:
$ mcpshell agent config show
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, err := initLogger()
		if err != nil {
			return err
		}

		// Get the config file path
		mcpShellHome, err := utils.GetMCPShellHome()
		if err != nil {
			return fmt.Errorf("failed to get MCPShell home directory: %w", err)
		}
		configPath := filepath.Join(mcpShellHome, "agent.yaml")

		// Load the current configuration
		config, err := agent.GetConfig()
		if err != nil {
			logger.Error("Failed to load config: %v", err)
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Check if config is empty
		if len(config.Agent.Models) == 0 {
			fmt.Printf("Configuration file: %s\n", configPath)
			fmt.Println()
			fmt.Println("No agent configuration found.")
			fmt.Println("Run 'mcpshell agent config create' to create a default configuration.")
			return nil
		}

		// Pretty print the configuration
		fmt.Printf("Configuration file: %s\n", configPath)
		fmt.Println()
		fmt.Println("Agent Configuration:")
		fmt.Println("===================")
		fmt.Println()

		for i, model := range config.Agent.Models {
			fmt.Printf("Model %d:\n", i+1)
			fmt.Printf("  Name: %s\n", model.Name)
			fmt.Printf("  Model: %s\n", model.Model)
			fmt.Printf("  Class: %s\n", model.Class)
			fmt.Printf("  Default: %t\n", model.Default)

			if model.APIKey != "" {
				if model.APIKey == "${OPENAI_API_KEY}" {
					fmt.Printf("  API Key: %s (from environment)\n", model.APIKey)
				} else {
					fmt.Printf("  API Key: %s\n", maskAPIKey(model.APIKey))
				}
			}

			if model.APIURL != "" {
				fmt.Printf("  API URL: %s\n", model.APIURL)
			}

			// Display prompts information
			if model.Prompts.HasSystemPrompts() {
				systemPrompts := model.Prompts.GetSystemPrompts()
				fmt.Printf("  System Prompts: %s\n", truncateString(systemPrompts, 80))
			}

			fmt.Println()
		}

		// Show which model is default
		defaultModel := config.GetDefaultModel()
		if defaultModel != nil {
			fmt.Printf("Default Model: %s (%s)\n", defaultModel.Name, defaultModel.Model)
		} else {
			fmt.Println("No default model configured.")
		}

		return nil
	},
}

// Helper function to mask API keys for security
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// Helper function to truncate long strings
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	// Add create and show subcommands to agent config
	agentConfigCommand.AddCommand(agentConfigCreateCommand)
	agentConfigCommand.AddCommand(agentConfigShowCommand)
}
