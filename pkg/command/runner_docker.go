// Package command provides functions for creating and executing command handlers.
package command

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/inercia/MCPShell/pkg/common"
)

// DockerRunner executes commands inside a Docker container.
type DockerRunner struct {
	logger *log.Logger
	opts   DockerRunnerOptions
}

// DockerRunnerOptions represents configuration options for the Docker runner.
type DockerRunnerOptions struct {
	// The Docker image to use (required)
	Image string `json:"image"`

	// Additional Docker run options
	DockerRunOpts string `json:"docker_run_opts"`

	// Mount points in the format "hostpath:containerpath"
	Mounts []string `json:"mounts"`

	// Whether to allow networking in the container
	AllowNetworking bool `json:"allow_networking"`

	// User to run as inside the container (defaults to current user)
	User string `json:"user"`

	// Working directory inside the container
	WorkDir string `json:"workdir"`
}

// parseDockerOptions extracts Docker-specific options from generic runner options.
func parseDockerOptions(genericOpts RunnerOptions) (DockerRunnerOptions, error) {
	opts := DockerRunnerOptions{
		AllowNetworking: true, // Default to allowing networking
		User:            "",   // Default to Docker's default user
		WorkDir:         "",   // Default to Docker's default working directory
	}

	// Parse image (required)
	if image, ok := genericOpts["image"].(string); ok {
		opts.Image = image
	} else {
		return opts, fmt.Errorf("docker runner requires 'image' option")
	}

	// Parse optional docker run options
	if dockerRunOpts, ok := genericOpts["docker_run_opts"].(string); ok {
		opts.DockerRunOpts = dockerRunOpts
	}

	// Parse optional mounts
	if mounts, ok := genericOpts["mounts"].([]interface{}); ok {
		for _, m := range mounts {
			if mountStr, ok := m.(string); ok {
				opts.Mounts = append(opts.Mounts, mountStr)
			}
		}
	}

	// Parse networking option
	if allowNetworking, ok := genericOpts["allow_networking"].(bool); ok {
		opts.AllowNetworking = allowNetworking
	}

	// Parse user option
	if user, ok := genericOpts["user"].(string); ok {
		opts.User = user
	}

	// Parse working directory option
	if workDir, ok := genericOpts["workdir"].(string); ok {
		opts.WorkDir = workDir
	}

	return opts, nil
}

// NewDockerRunner creates a new Docker runner with the specified options.
func NewDockerRunner(options RunnerOptions, logger *log.Logger) (*DockerRunner, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required for DockerRunner")
	}

	dockerOpts, err := parseDockerOptions(options)
	if err != nil {
		return nil, err
	}

	// Check if docker executable exists
	if !common.CheckExecutableExists("docker") {
		return nil, fmt.Errorf("docker executable not found in PATH")
	}

	// Check if Docker daemon is running
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.ID}}", "--no-trunc", "--limit", "1")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker daemon is not running: %w", err)
	}

	return &DockerRunner{
		logger: logger,
		opts:   dockerOpts,
	}, nil
}

// Run executes the command using Docker.
func (r *DockerRunner) Run(ctx context.Context, shell string, cmd string, env []string, params map[string]interface{}, tmpfile bool) (string, error) {
	// Create a temporary script file
	scriptFile, err := r.createScriptFile(shell, cmd, env)
	if err != nil {
		return "", fmt.Errorf("failed to create script file: %w", err)
	}
	defer os.Remove(scriptFile)

	r.logger.Printf("Created temporary script file: %s", scriptFile)

	// Construct the docker run command
	dockerCmd, err := r.buildDockerCommand(scriptFile, env)
	if err != nil {
		return "", fmt.Errorf("failed to build docker command: %w", err)
	}

	r.logger.Printf("Running command in Docker: %s", dockerCmd)

	// Use the exec runner to execute the docker command
	execRunner, err := NewRunnerExec(RunnerOptions{}, r.logger)
	if err != nil {
		return "", fmt.Errorf("failed to create exec runner: %w", err)
	}

	// Run the docker command - we set tmpfile to false because dockerCmd is already a full command
	output, err := execRunner.Run(ctx, "sh", dockerCmd, nil, params, false)
	if err != nil {
		return "", fmt.Errorf("docker command execution failed: %w", err)
	}

	return output, nil
}

// createScriptFile writes the command to a temporary script file.
func (r *DockerRunner) createScriptFile(shell string, cmd string, env []string) (string, error) {
	tmpDir := os.TempDir()
	scriptFile := filepath.Join(tmpDir, fmt.Sprintf("mcpshell-docker-%d.sh", os.Getpid()))

	// Prepare script content
	content := "#!/bin/sh\n\n"

	// Add environment variables
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			content += fmt.Sprintf("export %s=%s\n", parts[0], parts[1])
		}
	}

	// Add the command
	content += "\n# Command to execute\n"
	if shell != "" {
		content += fmt.Sprintf("exec %s -c %q\n", shell, cmd)
	} else {
		content += fmt.Sprintf("exec sh -c %q\n", cmd)
	}

	// Write the script
	if err := ioutil.WriteFile(scriptFile, []byte(content), 0755); err != nil {
		return "", err
	}

	return scriptFile, nil
}

// buildDockerCommand constructs the docker run command.
func (r *DockerRunner) buildDockerCommand(scriptFile string, env []string) (string, error) {
	// Basic docker run command
	parts := []string{"docker run --rm"}

	// Add networking option
	if !r.opts.AllowNetworking {
		parts = append(parts, "--network none")
	}

	// Add user if specified
	if r.opts.User != "" {
		parts = append(parts, fmt.Sprintf("--user %s", r.opts.User))
	}

	// Add working directory if specified
	if r.opts.WorkDir != "" {
		parts = append(parts, fmt.Sprintf("--workdir %s", r.opts.WorkDir))
	}

	// Add custom docker run options
	if r.opts.DockerRunOpts != "" {
		parts = append(parts, r.opts.DockerRunOpts)
	}

	// Mount the script file
	scriptName := filepath.Base(scriptFile)
	containerScriptPath := filepath.Join("/tmp", scriptName)
	parts = append(parts, fmt.Sprintf("-v %s:%s", scriptFile, containerScriptPath))

	// Add additional mounts
	for _, mount := range r.opts.Mounts {
		parts = append(parts, fmt.Sprintf("-v %s", mount))
	}

	// Add environment variables
	for _, e := range env {
		parts = append(parts, fmt.Sprintf("-e %s", e))
	}

	// Add image and command
	parts = append(parts, r.opts.Image)
	parts = append(parts, fmt.Sprintf("sh %s", containerScriptPath))

	// Join all parts
	dockerCmd := strings.Join(parts, " ")
	return dockerCmd, nil
} 