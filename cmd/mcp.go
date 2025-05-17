package root

import (
	"fmt"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/config"
	"github.com/inercia/MCPShell/pkg/server"
	"github.com/spf13/cobra"
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
		logger, err := initLogger()
		if err != nil {
			return err
		}

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
			ConfigFile:         localConfigPath,
			Logger:             logger,
			Version:            version,
			Description:        description,
			DescriptionFile:    descriptionFile,
			DescriptionAdd:     descriptionAdd,
			DescriptionAddFile: descriptionAddFile,
		})

		return srv.Start()
	},
}

// init adds flags to the run command
func init() {
	rootCmd.AddCommand(mcpCommand)

	// Add MCP-specific flags
	mcpCommand.Flags().StringVarP(&description, "description", "d", "", "MCP server description (optional)")
	mcpCommand.Flags().StringVarP(&descriptionFile, "description-file", "", "", "Read the MCP server description from a file (optional)")
	mcpCommand.Flags().StringVarP(&descriptionAdd, "description-add", "", "", "Add the given description to the MCP server description (optional)")
	mcpCommand.Flags().StringVarP(&descriptionAddFile, "description-add-file", "", "", "Read some additional text to add to the MCP server description from a file (optional)")

	// Mark required flags
	_ = mcpCommand.MarkFlagRequired("config")
}
