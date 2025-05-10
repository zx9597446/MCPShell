package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/exec"
	"strings"
)

// RunnerExec implements the Runner interface
type RunnerExec struct {
	logger *log.Logger
}

// RunnerExecOptions is the options for the RunnerExec
type RunnerExecOptions struct {
	Shell string `json:"shell"`
}

// NewRunnerExecOptions creates a new RunnerExecOptions from a RunnerOptions
func NewRunnerExecOptions(options RunnerOptions) (RunnerExecOptions, error) {
	var reopts RunnerExecOptions
	opts, err := options.ToJSON()
	if err != nil {
		return RunnerExecOptions{}, err
	}
	err = json.Unmarshal([]byte(opts), &reopts)
	return reopts, err
}

// NewRunnerExec creates a new ExecRunner with the provided logger
// If logger is nil, a default logger is created
func NewRunnerExec(logger *log.Logger) (*RunnerExec, error) {
	if logger == nil {
		logger = log.New(os.Stderr, "runner-exec: ", log.LstdFlags)
	}
	return &RunnerExec{
		logger: logger,
	}, nil
}

// Run executes a command with the given shell and returns the output
// It implements the Runner interface
func (r *RunnerExec) Run(ctx context.Context, shell string, command string, args []string, env []string, options RunnerOptions) (string, error) {
	// Combine command and args
	fullCmd := command
	if len(args) > 0 {
		fullCmd += " " + strings.Join(args, " ")
	}

	// Check if context is done
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue execution
	}

	// Set up the command
	configShell := getShell(shell)

	r.logger.Printf("Using shell: %s", configShell)

	// Create the command
	execCmd := exec.Command(configShell, "-c", fullCmd)

	r.logger.Printf("Created command: %s -c %s", configShell, fullCmd)

	// Set environment variables if provided
	if len(env) > 0 {
		r.logger.Printf("Adding %d environment variables to command", len(env))
		for _, e := range env {
			r.logger.Printf("... adding environment variable: %s", e)
		}
		execCmd.Env = append(os.Environ(), env...)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	// Run the command
	r.logger.Printf("Executing command")

	err := execCmd.Run()
	if err != nil {
		// If there's error output, include it in the error
		if stderr.Len() > 0 {
			errMsg := strings.TrimSpace(stderr.String())
			r.logger.Printf("Command failed with stderr: %s", errMsg)
			return "", errors.New(errMsg)
		}
		r.logger.Printf("Command failed with error: %v", err)
		return "", err
	}

	// Get the output
	output := strings.TrimSpace(stdout.String())

	r.logger.Printf("Command executed successfully, output length: %d bytes", len(output))
	if stderr.Len() > 0 {
		r.logger.Printf("Command generated stderr (but no error): %s", strings.TrimSpace(stderr.String()))
	}

	// Return the stdout output
	return output, nil
}

// getShell returns the shell to use for command execution,
// using the provided shell, falling back to $SHELL env var,
// and finally using /bin/sh as a last resort.
//
// Parameters:
//   - configShell: The configured shell to use (can be empty)
//
// Returns:
//   - The shell executable path to use
func getShell(configShell string) string {
	if configShell != "" {
		return configShell
	}

	shell := os.Getenv("SHELL")
	if shell != "" {
		return shell
	}

	return "/bin/sh"
}
