// Package root contains the command-line interface implementation for the MCP CLI Adapter.
//
// It defines the root command and all subcommands using Cobra and manages CLI flags,
// execution flow, and global application state.
package root

import (
	"fmt"
	"os"

	"github.com/inercia/mcp-cli-adapter/pkg/common"
	"github.com/spf13/cobra"
)

// ApplicationName is the name of the application used in various places
const ApplicationName = "mcp-cli-adapter"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   ApplicationName,
	Short: "MCP CLI Adapter",
	Long: `MCP CLI Adapter is a command line interface for the MCP platform.
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
	// Global flags can be set here
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.mcp-cli-adapter.yaml)")

	// Add version flag to all commands
	rootCmd.PersistentFlags().BoolP("version", "v", false, "Print version information")

	// Add commands
	rootCmd.AddCommand(runCommand)
	rootCmd.AddCommand(validateCommand)
}
