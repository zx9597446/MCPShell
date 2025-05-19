// Package root contains the command-line interface implementation for the MCPShell.
//
// It defines the root command and all subcommands using Cobra and manages CLI flags,
// execution flow, and global application state.
package root

import (
	"fmt"
	"os"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/spf13/cobra"
)

// ApplicationName is the name of the application used in various places
const ApplicationName = "mcpshell"

// Common command-line flags
var (
	// Common flags
	configFile string
	logFile    string
	logLevel   string

	// MCP server flags
	description         []string
	descriptionFile     []string
	descriptionOverride bool

	// Agent-specific flags
	agentModel        string
	agentSystemPrompt string
	agentUserPrompt   string
	agentOpenAIApiKey string
	agentOpenAIApiURL string
	agentOnce         bool

	// Application version (can be overridden at build time)
	version = "1.0.0"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   ApplicationName,
	Short: "MCPShell",
	Long: `MCPShell is a command line interface for the MCP platform.
This CLI application enables AI systems to securely execute commands through
the Model Context Protocol (MCP).`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is specified, show the help
		_ = cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	defer common.RecoverPanic()

	if err := rootCmd.Execute(); err != nil {
		common.GetLogger().Error("Command execution failed: %v", err)
		fmt.Println(err)
		os.Exit(1)
	}
}

// init registers all subcommands and sets up global flags
func init() {
	// Add common persistent flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to the YAML configuration file or URL")
	rootCmd.PersistentFlags().StringVarP(&logFile, "logfile", "l", "", "Path to the log file (optional)")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "", "info", "Log level: none, error, info, debug")

	// Add version flag to all commands
	rootCmd.PersistentFlags().BoolP("version", "v", false, "Print version information")
}

// initLogger initializes the logger with the specified configuration
func initLogger() (*common.Logger, error) {
	level := common.LogLevelFromString(logLevel)
	logger, err := common.NewLogger("[mcpshell] ", logFile, level, true)
	if err != nil {
		return nil, fmt.Errorf("failed to set up logger: %w", err)
	}

	common.SetLogger(logger)
	return logger, nil
}
