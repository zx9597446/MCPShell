package command

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/inercia/MCPShell/pkg/common"
)

//go:embed runner_firejail_profile.tpl
var firejailProfileTemplate string

// RunnerFirejail implements the Runner interface using firejail on Linux
type RunnerFirejail struct {
	logger     *log.Logger
	profileTpl *template.Template
	options    RunnerFirejailOptions
}

// RunnerFirejailOptions is the options for the RunnerFirejail
type RunnerFirejailOptions struct {
	Shell             string   `json:"shell"`
	AllowNetworking   bool     `json:"allow_networking"`
	AllowUserFolders  bool     `json:"allow_user_folders"`
	AllowReadFolders  []string `json:"allow_read_folders"`
	AllowWriteFolders []string `json:"allow_write_folders"`
	CustomProfile     string   `json:"custom_profile"`
}

// NewRunnerFirejailOptions creates a new RunnerFirejailOptions from a RunnerOptions
func NewRunnerFirejailOptions(options RunnerOptions) (RunnerFirejailOptions, error) {
	var reopts RunnerFirejailOptions
	opts, err := options.ToJSON()
	if err != nil {
		return RunnerFirejailOptions{}, err
	}
	err = json.Unmarshal([]byte(opts), &reopts)
	return reopts, err
}

// NewRunnerFirejail creates a new RunnerFirejail with the provided logger
// If logger is nil, a default logger is created
func NewRunnerFirejail(options RunnerOptions, logger *log.Logger) (*RunnerFirejail, error) {
	if logger == nil {
		logger = log.New(os.Stderr, "runner-firejail: ", log.LstdFlags)
	}

	// Parse the firejail profile template
	profileTpl, err := template.New("firejail-profile").Parse(firejailProfileTemplate)
	if err != nil {
		logger.Printf("Failed to parse firejail profile template: %v", err)
		return nil, err
	}

	// Parse firejail-specific options
	firejailOpts, err := NewRunnerFirejailOptions(options)
	if err != nil {
		logger.Printf("Failed to parse firejail options: %v", err)
		return nil, fmt.Errorf("failed to parse firejail options: %w", err)
	}

	return &RunnerFirejail{
		logger:     logger,
		profileTpl: profileTpl,
		options:    firejailOpts,
	}, nil
}

// Run executes a command inside the firejail sandbox and returns the output
// It implements the Runner interface
//
// note: tmpfile is ignored for firejail because it's not supported
func (r *RunnerFirejail) Run(ctx context.Context,
	shell string, command string,
	env []string, params map[string]interface{}, tmpfile bool,
) (string, error) {
	// Combine command and args
	fullCmd := command

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
		r.logger.Printf("Failed to render firejail profile template: %v", err)
		return "", fmt.Errorf("failed to render firejail profile: %w", err)
	}

	profile := profileBuf.String()
	r.logger.Printf("Firejail options: %+v", r.options)
	r.logger.Printf("Generated firejail profile: %s", profile)

	// Create a temporary file for the firejail profile
	profileFile, err := os.CreateTemp("", "firejail-profile-*.profile")
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

	// we always create a temporary file in the firejail environment
	// because the command might be executed with a different shell
	// and we need to ensure it's executable
	tmpFile, err := os.CreateTemp("", "firejail-command-*.sh")
	if err != nil {
		r.logger.Printf("Failed to create temporary command file: %v", err)
		return "", fmt.Errorf("failed to create temporary command file: %w", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			r.logger.Printf("Warning: failed to remove temporary command file: %v", err)
		}
	}()

	// write the command to the temporary file
	if _, err := tmpFile.WriteString(fullCmd); err != nil {
		r.logger.Printf("Failed to write command to temporary file: %v", err)
		return "", fmt.Errorf("failed to write command to temporary file: %w", err)
	}

	// make the temporary file executable
	if err := os.Chmod(tmpFile.Name(), 0o700); err != nil {
		r.logger.Printf("Failed to make temporary file executable: %v", err)
		return "", fmt.Errorf("failed to make temporary file executable: %w", err)
	}

	// Create the firejail command using --profile flag for file-based profile
	firejailCmd := fmt.Sprintf("firejail --profile=%s %s", profileFile.Name(), tmpFile.Name())
	r.logger.Printf("Created firejail command: %s", firejailCmd)

	// create a temporary file for the command
	tmpScript, err := os.CreateTemp("", "sandbox-script-*.sh")
	if err != nil {
		r.logger.Printf("Failed to create temporary command file: %v", err)
		return "", fmt.Errorf("failed to create temporary command file: %w", err)
	}
	// Ensure temporary file is deleted when this function exits
	defer func() {
		tmpScriptPath := tmpScript.Name()
		if err := tmpScript.Close(); err != nil {
			r.logger.Printf("Warning: failed to close profile file: %v", err)
		}
		if err := os.Remove(tmpScriptPath); err != nil {
			r.logger.Printf("Warning: failed to remove temporary profile file: %v", err)
		}
	}()

	// write the command to the temporary file
	// we always create a temporary file in the sandboxed environment
	if _, err := tmpScript.WriteString(fullCmd); err != nil {
		r.logger.Printf("Failed to write command to temporary file: %v", err)
		return "", fmt.Errorf("failed to write command to temporary file: %w", err)
	}

	// Flush data to ensure it's written to disk
	if err := tmpScript.Sync(); err != nil {
		r.logger.Printf("Failed to sync script file: %v", err)
		return "", fmt.Errorf("failed to sync script file: %w", err)
	}

	// make the temporary file executable
	if err := os.Chmod(tmpScript.Name(), 0o700); err != nil {
		r.logger.Printf("Failed to make temporary file executable: %v", err)
		return "", fmt.Errorf("failed to make temporary file executable: %w", err)
	}

	// Check if context is done
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue execution
	}

	// Execute the command directly without a temporary file
	execCmd := exec.Command("firejail", "--profile="+profileFile.Name(), tmpFile.Name())
	r.logger.Printf("Created command: %s", execCmd.String())

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

	if err := execCmd.Run(); err != nil {
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
	outputStr := strings.TrimSpace(stdout.String())

	r.logger.Printf("Command executed successfully, output length: %d bytes", len(outputStr))
	if stderr.Len() > 0 {
		r.logger.Printf("Command generated stderr (but no error): %s", strings.TrimSpace(stderr.String()))
	}

	// Return the stdout output
	return outputStr, nil
}
