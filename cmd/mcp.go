package root

import (
	"fmt"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/config"
	"github.com/inercia/MCPShell/pkg/server"
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

// mcpCommand represents the run command which starts the MCP server
var mcpCommand = &cobra.Command{
	Use:     "mcp",
	Aliases: []string{"serve", "server", "run"},
	Short:   "Run the MCP server for a MCP configuration file",
	Long: `
Run an MCP server that provides tools to LLM applications.
This command starts a server that communicates using the Model Context Protocol (MCP)
and expooses the tools defined in a MCP configuration file.

The server loads tool definitions from a YAML configuration file and makes them
available to AI applications via the MCP protocol.
`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger
		level := common.LogLevelFromString(logLevel)
		logger, err := common.NewLogger("[mcpshell] ", logFile, level, true)
		if err != nil {
			return fmt.Errorf("failed to set up logger: %w", err)
		}

		common.SetLogger(logger)
		defer common.RecoverPanic()

		logger.Info("Starting MCPShell")

		// Check if config file is provided
		if configFile == "" {
			logger.Error("Configuration file is required")
			return fmt.Errorf("configuration file is required. Use --config or -c flag to specify the path")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := common.GetLogger()
		defer common.RecoverPanic()

		// Load the configuration file (local or remote)
		localConfigPath, cleanup, err := config.ResolveConfigPath(configFile, logger)
		if err != nil {
			logger.Error("Failed to load configuration: %v", err)
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Ensure temporary files are cleaned up
		defer cleanup()

		// Create and start the server
		srv := server.New(server.Config{
			ConfigFile:  localConfigPath,
			Logger:      logger,
			Version:     version,
			Description: description,
		})

		return srv.Start()
	},
}

// init adds flags to the run command
func init() {
	rootCmd.AddCommand(mcpCommand)

	// Add flags for the run command
	mcpCommand.Flags().StringVarP(&configFile, "config", "c", "", "Path to the YAML configuration file or URL (required)")
	mcpCommand.Flags().StringVarP(&logFile, "logfile", "l", "", "Path to the log file (optional)")
	mcpCommand.Flags().StringVarP(&logLevel, "log-level", "", "info", "Log level: none, error, info, debug")
	mcpCommand.Flags().StringVarP(&description, "description", "d", "", "Server description (optional)")

	// Mark required flags
	_ = mcpCommand.MarkFlagRequired("config")
}
