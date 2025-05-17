package root

import (
	"fmt"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/config"
	"github.com/inercia/MCPShell/pkg/server"
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
		logger, err := initLogger()
		if err != nil {
			return err
		}

		// Setup panic handler
		defer func() {
			if logger != nil {
				common.RecoverPanic()
			}
		}()

		logger.Info("Validating MCP configuration")

		// Check if config file is provided
		if configFile == "" {
			logger.Error("Configuration file is required")
			return fmt.Errorf("configuration file is required. Use --config or -c flag to specify the path")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the logger
		logger := common.GetLogger()

		// Setup panic handler
		defer func() {
			if logger != nil {
				common.RecoverPanic()
			}
		}()

		// Load the configuration file (local or remote)
		localConfigPath, cleanup, err := config.ResolveConfigPath(configFile, logger)
		if err != nil {
			logger.Error("Failed to load configuration: %v", err)
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Ensure temporary files are cleaned up
		defer cleanup()

		// Create server instance for validation only
		srv := server.New(server.Config{
			ConfigFile:  localConfigPath,
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

	// Mark required flags
	_ = validateCommand.MarkFlagRequired("config")
}
