package root

import (
	"fmt"
	"os"

	"github.com/inercia/mcp-cli-adapter/pkg/common"
	"github.com/inercia/mcp-cli-adapter/pkg/server"
	"github.com/spf13/cobra"
)

// validateCommand represents the validate command which checks a configuration file
var validateCommand = &cobra.Command{
	Use:   "validate",
	Short: "Validate an MCP configuration file",
	Long: `Validate an MCP configuration file without starting the server.
This command checks the configuration file for errors including:
- File format and schema validation
- Tool parameter definitions
- Constraint expression syntax
- Command template syntax`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger
		logger, err := setupLogger()
		if err != nil {
			return fmt.Errorf("failed to set up logger: %w", err)
		}

		// Set global logger for application-wide use
		SetLogger(logger)

		// Setup panic handler
		defer func() {
			if logger != nil {
				common.RecoverPanic(logger.Logger, logger.FilePath())
			}
		}()

		logger.Info("Validating MCP configuration")

		// Check if config file is provided
		if configFile == "" {
			logger.Error("Configuration file is required")
			return fmt.Errorf("configuration file is required. Use --config or -c flag to specify the path")
		}

		// Check if the config file exists
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			logger.Error("Configuration file does not exist: %s", configFile)
			return fmt.Errorf("configuration file does not exist: %s", configFile)
		}

		logger.Info("Using configuration file: %s", configFile)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the logger
		logger := GetLogger()

		// Setup panic handler
		defer func() {
			if logger != nil {
				common.RecoverPanic(logger.Logger, logger.FilePath())
			}
		}()

		// Create server instance for validation only
		srv := server.New(server.Config{
			ConfigFile:  configFile,
			Logger:      logger,
			Version:     version,
			Description: description,
		})

		// Validate the configuration
		if err := srv.Validate(); err != nil {
			logger.Error("Configuration validation failed: %v", err)
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		logger.Info("Configuration validation successful")
		return nil
	},
}

// init adds the validate command to the root command
func init() {
	// Add validate command to root
	rootCmd.AddCommand(validateCommand)

	// Add the same flags as the run command
	validateCommand.Flags().StringVarP(&configFile, "config", "c", "", "Path to the YAML configuration file (required)")
	validateCommand.Flags().StringVarP(&logFile, "logfile", "l", "", "Path to the log file (optional)")
	validateCommand.Flags().StringVarP(&logLevel, "log-level", "", "info", "Log level: none, error, info, debug")

	// Mark required flags
	_ = validateCommand.MarkFlagRequired("config")
}
