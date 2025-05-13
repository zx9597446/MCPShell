// Package main provides the entry point for the MCPShell application.
//
// The application implements the Model Context Protocol (MCP) for executing
// command-line tools in a secure and configurable manner, allowing AI-powered
// applications to execute commands on behalf of users.
package main

import (
	cmdroot "github.com/inercia/MCPShell/cmd"
	"github.com/inercia/MCPShell/pkg/common"
)

// main is the entry point of the application. It sets up the panic recovery system
// at the top level and executes the root command, which will process CLI flags and
// execute the selected subcommand.
func main() {
	// Setup global panic recovery that will catch any unhandled panics
	// and prevent the application from crashing uncleanly
	defer func() {
		common.RecoverPanic()
	}()

	// Execute the root command
	cmdroot.Execute()
}
