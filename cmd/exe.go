package root

import (
	"fmt"
	"strings"

	"github.com/inercia/MCPShell/pkg/command"
	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/config"
	"github.com/spf13/cobra"
)

// exeConfigFile is the path to the configuration file
var exeConfigFile string

// exeLogFile is the path to the log file
var exeLogFile string

// exeLogLevel is the logging level
var exeLogLevel string

// exeCommand is a command that executes a MCP tool
var exeCommand = &cobra.Command{
	Use:   "exe",
	Short: "Execute a MCP tool",
	Long: `
Execution of a MCP tool.

This command will execute a MCP tool. Sometimes it is difficult to debug
the execution of a MCP tool. This command will help you to debug the tool
by executing it with the given parameters, following the whole process
of constraint evaluation, tool selection and tool execution.

For example, you can run:

$ mcpshell exe --configfile=examples/config.yaml "hello_world" "name=John"

and it will run the "hello_world" tool with the parameter "name" set to "John".
Any error in the constraint evaluation, tool selection or tool execution
will be reported.

`,
	Args: cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger
		level := common.LogLevelFromString(exeLogLevel)
		logger, err := common.NewLogger("[mcpshell] ", exeLogFile, level, true)
		if err != nil {
			return fmt.Errorf("failed to set up logger: %w", err)
		}

		// Set global logger for application-wide use
		common.SetLogger(logger)

		// Setup panic handler
		defer common.RecoverPanic()

		logger.Info("Executing MCP tool directly")

		// Check if config file is provided
		if exeConfigFile == "" {
			logger.Error("Configuration file is required")
			return fmt.Errorf("configuration file is required. Use --config or -c flag to specify the path")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the logger
		logger := common.GetLogger()

		// Setup panic handler
		defer common.RecoverPanic()

		// Get the tool name
		toolName := args[0]
		logger.Info("Executing tool: %s", toolName)

		// Load the configuration file (local or remote)
		localConfigPath, cleanup, err := config.ResolveConfigPath(exeConfigFile, logger)
		if err != nil {
			logger.Error("Failed to load configuration: %v", err)
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Ensure temporary files are cleaned up
		defer cleanup()

		// Load the configuration
		cfg, err := config.NewConfigFromFile(localConfigPath)
		if err != nil {
			logger.Error("Failed to load configuration: %v", err)
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Find the requested tool in the configuration
		var targetTool *config.MCPToolConfig
		for _, toolConfig := range cfg.MCP.Tools {
			if toolConfig.Name == toolName {
				targetTool = &toolConfig
				break
			}
		}

		if targetTool == nil {
			logger.Error("Tool not found: %s", toolName)
			return fmt.Errorf("tool not found: %s", toolName)
		}

		// Parse parameters from the remaining arguments
		params := make(map[string]interface{})
		for _, arg := range args[1:] {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) != 2 {
				logger.Error("Invalid parameter format: %s (expected name=value)", arg)
				return fmt.Errorf("invalid parameter format: %s (expected name=value)", arg)
			}
			paramName := parts[0]
			paramValue := parts[1]

			// Check if parameter is defined in the tool
			paramConfig, exists := targetTool.Params[paramName]
			if !exists {
				logger.Error("Parameter not defined in tool: %s", paramName)
				return fmt.Errorf("parameter not defined in tool: %s", paramName)
			}

			// Convert parameter value to appropriate type based on parameter config
			typedValue, err := common.ConvertStringToType(paramValue, paramConfig.Type)
			if err != nil {
				logger.Error("Failed to convert parameter value: %v", err)
				return fmt.Errorf("failed to convert parameter value: %w", err)
			}

			params[paramName] = typedValue
		}

		// Check required parameters
		for paramName, paramConfig := range targetTool.Params {
			if paramConfig.Required {
				if _, exists := params[paramName]; !exists {
					logger.Error("Required parameter missing: %s", paramName)
					return fmt.Errorf("required parameter missing: %s", paramName)
				}
			}
		}

		// Use shell from config if present
		shell := cfg.MCP.Run.Shell
		if shell == "" {
			shell = "sh"
		}

		// Create a command handler
		handler, err := command.NewCommandHandler(config.Tool{
			MCPTool: config.CreateMCPTool(*targetTool),
			Config:  *targetTool,
		}, targetTool.Params, shell, logger.Logger)
		if err != nil {
			logger.Error("Failed to create command handler: %v", err)
			return fmt.Errorf("failed to create command handler: %w", err)
		}

		// Execute the command directly
		result, err := handler.ExecuteCommand(params)
		if err != nil {
			logger.Error("Command execution failed: %v", err)
			return fmt.Errorf("command execution failed: %w", err)
		}

		// Print the result
		fmt.Println(result)
		return nil
	},
}

// init adds the exe command to the root command
func init() {
	// Add exe command to root
	rootCmd.AddCommand(exeCommand)

	// Add flags
	exeCommand.Flags().StringVarP(&exeConfigFile, "config", "c", "", "Path to the YAML configuration file or URL (required)")
	exeCommand.Flags().StringVarP(&exeLogFile, "logfile", "l", "", "Path to the log file (optional)")
	exeCommand.Flags().StringVarP(&exeLogLevel, "log-level", "", "info", "Log level: none, error, info, debug")

	// Mark required flags
	_ = exeCommand.MarkFlagRequired("config")
}
