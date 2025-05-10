package root

import (
	"fmt"
	"os"

	"github.com/inercia/mcp-cli-adapter/pkg/common"
	"github.com/inercia/mcp-cli-adapter/pkg/server"
	"github.com/spf13/cobra"
)

var (
	// Command-line flags
	configFile  string
	logFile     string
	logLevel    string
	description string

	// Application version (can be overridden at build time)
	version = "1.0.0"
)

// runCommand represents the run command which starts the MCP server
var runCommand = &cobra.Command{
	Use:   "run",
	Short: "Run an MCP server",
	Long: `Run an MCP server that provides tools to LLM applications.
This command starts a server that communicates using the Model Context Protocol (MCP).

The server loads tool definitions from a YAML configuration file and makes them
available to AI applications via the MCP protocol.`,
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

		logger.Info("Starting MCP CLI Adapter")

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

		// Create and start the server
		srv := server.New(server.Config{
			ConfigFile:  configFile,
			Logger:      logger,
			Version:     version,
			Description: description,
		})

		return srv.Start()
	},
}

// init adds flags to the run command
func init() {
	// Add flags for the run command
	runCommand.Flags().StringVarP(&configFile, "config", "c", "", "Path to the YAML configuration file (required)")
	runCommand.Flags().StringVarP(&logFile, "logfile", "l", "", "Path to the log file (optional)")
	runCommand.Flags().StringVarP(&logLevel, "log-level", "", "info", "Log level: none, error, info, debug")
	runCommand.Flags().StringVarP(&description, "description", "d", "", "Server description (optional)")

	// Mark required flags
	_ = runCommand.MarkFlagRequired("config")
}
