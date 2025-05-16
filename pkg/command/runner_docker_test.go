package command

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/inercia/MCPShell/pkg/common"
)

// checkDockerRunning verifies that Docker is installed and the daemon is running
func checkDockerRunning() bool {
	// First check if Docker executable exists
	if !common.CheckExecutableExists("docker") {
		return false
	}

	// Then try to run a simple docker command to check if the daemon is running
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "stats", "--no-stream")
	err := cmd.Run()
	return err == nil
}

func TestDockerRunnerInitialization(t *testing.T) {
	if !checkDockerRunning() {
		t.Skip("Docker not installed or not running, skipping test")
	}

	logger := log.New(os.Stderr, "test-docker: ", log.LstdFlags)

	testCases := []struct {
		name        string
		options     RunnerOptions
		expectError bool
	}{
		{
			name: "Valid options",
			options: RunnerOptions{
				"image": "alpine:latest",
			},
			expectError: false,
		},
		{
			name:        "Missing image",
			options:     RunnerOptions{},
			expectError: true,
		},
		{
			name: "Full options",
			options: RunnerOptions{
				"image":            "ubuntu:latest",
				"allow_networking": false,
				"mounts":           []interface{}{"/tmp:/tmp"},
				"user":             "nobody",
				"workdir":          "/app",
				"docker_run_opts":  "--cpus 0.5",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewDockerRunner(tc.options, logger)
			if tc.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestDockerRunnerBasic(t *testing.T) {
	// Skip if docker is not available or not running
	if !checkDockerRunning() {
		t.Skip("Docker not installed or not running, skipping test")
	}

	logger := log.New(os.Stderr, "test-docker: ", log.LstdFlags)

	// Create a runner with alpine image
	runner, err := NewDockerRunner(RunnerOptions{
		"image": "alpine:latest",
	}, logger)

	if err != nil {
		t.Fatalf("Failed to create Docker runner: %v", err)
	}

	// Test a simple echo command
	output, err := runner.Run(context.Background(), "", "echo 'Hello from Docker'", nil, nil, false)
	if err != nil {
		t.Errorf("Failed to run command: %v", err)
	}

	// Check the output
	expected := "Hello from Docker"
	if output != expected {
		t.Errorf("Expected output %q, got %q", expected, output)
	}
}

func TestDockerRunnerNetworking(t *testing.T) {
	// Skip if docker is not available or not running
	if !checkDockerRunning() {
		t.Skip("Docker not installed or not running, skipping test")
	}

	logger := log.New(os.Stderr, "test-docker: ", log.LstdFlags)

	testCases := []struct {
		name            string
		allowNetworking bool
		expectSuccess   bool
	}{
		{
			name:            "With networking",
			allowNetworking: true,
			expectSuccess:   true,
		},
		{
			name:            "Without networking",
			allowNetworking: false,
			expectSuccess:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a runner with specified networking
			runner, err := NewDockerRunner(RunnerOptions{
				"image":            "alpine:latest",
				"allow_networking": tc.allowNetworking,
			}, logger)

			if err != nil {
				t.Fatalf("Failed to create Docker runner: %v", err)
			}

			// Try to ping google.com (will fail if networking is disabled)
			_, err = runner.Run(context.Background(), "", "ping -c 1 -W 1 google.com", nil, nil, false)

			if tc.expectSuccess && err != nil {
				t.Errorf("Expected network ping to succeed but got error: %v", err)
			}

			if !tc.expectSuccess && err == nil {
				t.Errorf("Expected network ping to fail but it succeeded")
			}
		})
	}
}

func TestDockerRunnerEnvironmentVariables(t *testing.T) {
	// Skip if docker is not available or not running
	if !checkDockerRunning() {
		t.Skip("Docker not installed or not running, skipping test")
	}

	logger := log.New(os.Stderr, "test-docker: ", log.LstdFlags)

	// Create a runner with alpine image
	runner, err := NewDockerRunner(RunnerOptions{
		"image": "alpine:latest",
	}, logger)

	if err != nil {
		t.Fatalf("Failed to create Docker runner: %v", err)
	}

	// Define environment variables to pass to the container
	env := []string{
		"TEST_VAR1=test_value1",
		"TEST_VAR2=test_value2",
		"TEST_VAR3=value_with_underscores",
	}

	// Run a command that echoes the environment variables
	output, err := runner.Run(context.Background(), "", "echo $TEST_VAR1,$TEST_VAR2,$TEST_VAR3", env, nil, false)
	if err != nil {
		t.Errorf("Failed to run command with environment variables: %v", err)
	}

	// Check the output contains the environment variable values
	expected := "test_value1,test_value2,value_with_underscores"
	if output != expected {
		t.Errorf("Environment variables not correctly passed. Expected %q, got %q", expected, output)
	}

	// Test with a mix of shell variables and environment variables
	output, err = runner.Run(context.Background(), "sh", "echo $TEST_VAR1 and $TEST_VAR2", env, nil, false)
	if err != nil {
		t.Errorf("Failed to run command with mixed variables: %v", err)
	}

	// Check that at least the environment variables are included in the output
	if !strings.Contains(output, "test_value1") || !strings.Contains(output, "test_value2") {
		t.Errorf("Environment variables not found in output with shell variables: %q", output)
	}
}

func TestDockerRunnerPrepareCommand(t *testing.T) {
	// Skip if docker is not available or not running
	if !checkDockerRunning() {
		t.Skip("Docker not installed or not running, skipping test")
	}

	logger := log.New(os.Stderr, "test-docker: ", log.LstdFlags)

	// Create a runner with alpine image and prepare command to install grep
	runner, err := NewDockerRunner(RunnerOptions{
		"image":           "alpine:latest",
		"prepare_command": "apk add --no-cache grep",
	}, logger)

	if err != nil {
		t.Fatalf("Failed to create Docker runner: %v", err)
	}

	// Run grep command that should only work if the prepare_command executed properly
	output, err := runner.Run(context.Background(), "", "grep --version | head -n 1", nil, nil, false)
	if err != nil {
		t.Errorf("Failed to run command that requires prepare_command: %v", err)
	}

	// Check the output contains grep version information
	if !strings.Contains(output, "grep") {
		t.Errorf("Expected output to contain grep version information, got: %q", output)
	}
}
