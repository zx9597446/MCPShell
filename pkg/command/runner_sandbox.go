package command

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/inercia/MCPShell/pkg/common"
)

//go:embed runner_sandbox_profile.tpl
var sandboxProfileTemplate string

// RunnerSandboxExec implements the Runner interface using macOS sandbox-exec
type RunnerSandboxExec struct {
	execRunner *RunnerExec
	logger     *log.Logger
	profileTpl *template.Template
	options    RunnerSandboxExecOptions
}

// RunnerSandboxExecOptions is the options for the RunnerSandboxExec
type RunnerSandboxExecOptions struct {
	Shell             string   `json:"shell"`
	AllowNetworking   bool     `json:"allow_networking"`
	AllowUserFolders  bool     `json:"allow_user_folders"`
	AllowReadFolders  []string `json:"allow_read_folders"`
	AllowWriteFolders []string `json:"allow_write_folders"`
	CustomProfile     string   `json:"custom_profile"`
}

// NewRunnerSandboxExecOptions creates a new RunnerSandboxExecOptions from a RunnerOptions
func NewRunnerSandboxExecOptions(options RunnerOptions) (RunnerSandboxExecOptions, error) {
	var reopts RunnerSandboxExecOptions
	opts, err := options.ToJSON()
	if err != nil {
		return RunnerSandboxExecOptions{}, err
	}
	err = json.Unmarshal([]byte(opts), &reopts)
	return reopts, err
}

// NewRunnerSandboxExec creates a new RunnerSandboxExec with the provided logger
// If logger is nil, a default logger is created
func NewRunnerSandboxExec(options RunnerOptions, logger *log.Logger) (*RunnerSandboxExec, error) {
	if logger == nil {
		logger = log.New(os.Stderr, "runner-sandbox-exec: ", log.LstdFlags)
	}

	// Parse the sandbox profile template
	profileTpl, err := template.New("sandbox-profile").Parse(sandboxProfileTemplate)
	if err != nil {
		logger.Printf("Failed to parse sandbox profile template: %v", err)
		return nil, err
	}

	execRunner, err := NewRunnerExec(options, logger)
	if err != nil {
		return nil, err
	}

	// Parse sandbox-specific options
	sandboxOpts, err := NewRunnerSandboxExecOptions(options)
	if err != nil {
		logger.Printf("Failed to parse sandbox options: %v", err)
		return nil, fmt.Errorf("failed to parse sandbox options: %w", err)
	}

	return &RunnerSandboxExec{
		execRunner: execRunner,
		logger:     logger,
		profileTpl: profileTpl,
		options:    sandboxOpts,
	}, nil
}

// Run executes a command inside the macOS sandbox and returns the output
// It implements the Runner interface
func (r *RunnerSandboxExec) Run(ctx context.Context, shell string, command string, args []string, env []string, params map[string]interface{}) (string, error) {
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

	// replace template variables in allow read and write folders
	if len(r.options.AllowReadFolders) > 0 {
		r.options.AllowReadFolders = common.ProcessTemplateListFlexible(r.options.AllowReadFolders, params)
	}
	if len(r.options.AllowWriteFolders) > 0 {
		r.options.AllowWriteFolders = common.ProcessTemplateListFlexible(r.options.AllowWriteFolders, params)
	}

	// Generate the profile by rendering the template
	var profileBuf bytes.Buffer
	if err := r.profileTpl.Execute(&profileBuf, r.options); err != nil {
		r.logger.Printf("Failed to render sandbox profile template: %v", err)
		return "", fmt.Errorf("failed to render sandbox profile: %w", err)
	}

	profile := profileBuf.String()
	r.logger.Printf("Sandbox options: %+v", r.options)
	r.logger.Printf("Generated sandbox profile: %s", profile)

	// Create a temporary file for the sandbox profile
	profileFile, err := os.CreateTemp("", "sandbox-profile-*.sb")
	if err != nil {
		r.logger.Printf("Failed to create temporary profile file: %v", err)
		return "", fmt.Errorf("failed to create temporary profile file: %w", err)
	}

	// Ensure temporary file is deleted when this function exits
	defer func() {
		profileFilePath := profileFile.Name()
		if err := profileFile.Close(); err != nil {
			r.logger.Printf("Warning: failed to close profile file: %v", err)
		}
		if err := os.Remove(profileFilePath); err != nil {
			r.logger.Printf("Warning: failed to remove temporary profile file: %v", err)
		}
	}()

	// Write the profile to the temporary file
	if _, err := profileFile.WriteString(profile); err != nil {
		r.logger.Printf("Failed to write profile to temporary file: %v", err)
		return "", fmt.Errorf("failed to write profile to temporary file: %w", err)
	}

	// Flush data to ensure it's written to disk
	if err := profileFile.Sync(); err != nil {
		r.logger.Printf("Failed to sync profile file: %v", err)
		return "", fmt.Errorf("failed to sync profile file: %w", err)
	}

	// Create the sandboxed command using -f flag for file-based profile
	sandboxCmd := fmt.Sprintf("sandbox-exec -f %s %s", profileFile.Name(), fullCmd)
	r.logger.Printf("Created sandboxed command: %s", sandboxCmd)

	// Execute the command using the embedded RunnerExec
	return r.execRunner.Run(ctx, shell, sandboxCmd, []string{}, env, params)
}
