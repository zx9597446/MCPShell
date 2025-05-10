// Package main provides the entry point for the MCP CLI Adapter application.
//
// The application implements the Model Context Protocol (MCP) for executing
// command-line tools in a secure and configurable manner, allowing AI-powered
// applications to execute commands on behalf of users.
package main

import (
	cmdroot "github.com/inercia/mcp-cli-adapter/cmd"
	"github.com/inercia/mcp-cli-adapter/pkg/common"
)

// main is the entry point of the application. It sets up the panic recovery system
// at the top level and executes the root command, which will process CLI flags and
// execute the selected subcommand.
func main() {
	// Setup global panic recovery that will catch any unhandled panics
	// and prevent the application from crashing uncleanly
	defer func() {
		common.RecoverPanic(nil, "")
	}()

	// Execute the root command
	cmdroot.Execute()
}
