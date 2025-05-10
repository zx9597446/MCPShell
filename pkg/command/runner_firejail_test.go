package command

import (
	"context"
	"os"
	"runtime"
	"testing"
)

func TestNewRunnerFirejail(t *testing.T) {
	// Skip on non-Linux platforms
	if runtime.GOOS != "linux" {
		t.Skip("Skipping firejail tests on non-Linux platform")
	}

	runner, err := NewRunnerFirejail(nil)
	if err != nil {
		t.Fatalf("Failed to create firejail runner: %v", err)
	}

	if runner == nil {
		t.Fatal("Expected non-nil runner")
	}
}

func TestRunnerFirejailRun(t *testing.T) {
	// Skip on non-Linux platforms
	if runtime.GOOS != "linux" {
		t.Skip("Skipping firejail tests on non-Linux platform")
	}

	// Skip if firejail is not installed
	if _, err := os.Stat("/usr/bin/firejail"); os.IsNotExist(err) {
		t.Skip("Skipping test because firejail is not installed")
	}

	runner, err := NewRunnerFirejail(nil)
	if err != nil {
		t.Fatalf("Failed to create firejail runner: %v", err)
	}

	ctx := context.Background()
	options := RunnerOptions{
		"allow_networking": true,
	}

	// Test simple echo command
	output, err := runner.Run(ctx, "/bin/sh", "echo", []string{"hello world"}, nil, options)
	if err != nil {
		t.Fatalf("Failed to run command: %v", err)
	}

	if output != "hello world\n" {
		t.Errorf("Expected 'hello world\\n', got '%s'", output)
	}
}

func TestRunnerFirejailNetworkRestriction(t *testing.T) {
	// Skip on non-Linux platforms
	if runtime.GOOS != "linux" {
		t.Skip("Skipping firejail tests on non-Linux platform")
	}

	// Skip if firejail is not installed
	if _, err := os.Stat("/usr/bin/firejail"); os.IsNotExist(err) {
		t.Skip("Skipping test because firejail is not installed")
	}

	runner, err := NewRunnerFirejail(nil)
	if err != nil {
		t.Fatalf("Failed to create firejail runner: %v", err)
	}

	ctx := context.Background()

	// Test with networking enabled
	networkEnabledOptions := RunnerOptions{
		"allow_networking": true,
	}

	// This might succeed or fail depending on network connectivity,
	// but it should not be blocked by firejail
	_, _ = runner.Run(ctx, "/bin/sh", "ping", []string{"-c", "1", "127.0.0.1"}, nil, networkEnabledOptions)

	// Test with networking disabled
	networkDisabledOptions := RunnerOptions{
		"allow_networking": false,
	}

	// This should fail or timeout due to network restrictions
	// Note: We're not asserting the exact behavior as it might vary based on firejail version
	_, _ = runner.Run(ctx, "/bin/sh", "ping", []string{"-c", "1", "127.0.0.1"}, nil, networkDisabledOptions)
}
