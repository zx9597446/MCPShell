package command

import (
	"context"
	"reflect"
	"runtime"
	"testing"
)

func TestNewRunnerSandboxExecOptions(t *testing.T) {
	// Skip on non-macOS platforms
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	tests := []struct {
		name    string
		options RunnerOptions
		want    RunnerSandboxExecOptions
		wantErr bool
	}{
		{
			name: "valid options with all fields",
			options: RunnerOptions{
				"shell":              "/bin/bash",
				"allow_networking":   true,
				"allow_user_folders": true,
				"custom_profile":     "(version 1)(allow default)",
			},
			want: RunnerSandboxExecOptions{
				Shell:            "/bin/bash",
				AllowNetworking:  true,
				AllowUserFolders: true,
				CustomProfile:    "(version 1)(allow default)",
			},
			wantErr: false,
		},
		{
			name:    "empty options",
			options: RunnerOptions{},
			want:    RunnerSandboxExecOptions{},
			wantErr: false,
		},
		{
			name: "options with partial fields",
			options: RunnerOptions{
				"shell":            "/bin/zsh",
				"allow_networking": false,
			},
			want: RunnerSandboxExecOptions{
				Shell:           "/bin/zsh",
				AllowNetworking: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRunnerSandboxExecOptions(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRunnerSandboxExecOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRunnerSandboxExecOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

// This test is only run on macOS as it requires sandbox-exec
func TestRunnerSandboxExec_Run(t *testing.T) {
	// Skip on non-macOS platforms
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	// Also skip if the short flag is set
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	logger := testLogger
	runner, err := NewRunnerSandboxExec(logger)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}
	ctx := context.Background()
	shell := "" // use default

	tests := []struct {
		name          string
		command       string
		args          []string
		options       RunnerOptions
		shouldSucceed bool
		expectedOut   string
	}{
		{
			name:    "echo command with full permissions",
			command: "echo 'Hello Sandbox'",
			args:    []string{},
			options: RunnerOptions{
				"allow_networking":   true,
				"allow_user_folders": true,
			},
			shouldSucceed: true,
			expectedOut:   "Hello Sandbox",
		},
		{
			name:    "echo command with networking disabled",
			command: "echo 'No Network'",
			args:    []string{},
			options: RunnerOptions{
				"allow_networking":   false,
				"allow_user_folders": true,
			},
			shouldSucceed: true,
			expectedOut:   "No Network",
		},
		{
			name:    "echo command with all restrictions",
			command: "echo 'Restricted'",
			args:    []string{},
			options: RunnerOptions{
				"allow_networking":   false,
				"allow_user_folders": false,
			},
			shouldSucceed: true,
			expectedOut:   "Restricted",
		},
		{
			name:    "read /tmp with folder restrictions",
			command: "ls -la /tmp | grep -q . && echo 'success'",
			args:    []string{},
			options: RunnerOptions{
				"allow_networking":   false,
				"allow_user_folders": false,
			},
			shouldSucceed: true,
			expectedOut:   "success",
		},
		{
			name:    "custom profile allowing only /tmp",
			command: "ls -la /tmp | grep -q . && echo 'success'",
			args:    []string{},
			options: RunnerOptions{
				"custom_profile": `(version 1)
(allow default)
(deny file-read* (subpath "/Users"))
(allow file-read* (regex "^/tmp"))`,
			},
			shouldSucceed: true,
			expectedOut:   "success",
		},
		{
			name:    "custom profile blocking all except echo",
			command: "echo 'only echo works'",
			args:    []string{},
			options: RunnerOptions{
				"custom_profile": `(version 1)
(allow default)
(deny process-exec*)
(allow process-exec* (regex "^/bin/echo"))
(allow process-exec* (regex "^/usr/bin/echo"))`,
			},
			shouldSucceed: true,
			expectedOut:   "only echo works",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := runner.Run(ctx, shell, tt.command, tt.args, []string{}, tt.options)

			// Check if success/failure matches expectations
			if tt.shouldSucceed && err != nil {
				t.Errorf("Expected command to succeed but got error: %v", err)
				return
			}

			if !tt.shouldSucceed && err == nil {
				t.Errorf("Expected command to fail but it succeeded with output: %s", output)
				return
			}

			// If we should succeed and we have an expected output, check it
			if tt.shouldSucceed && tt.expectedOut != "" && output != tt.expectedOut {
				t.Errorf("Output mismatch: got %v, want %v", output, tt.expectedOut)
			}
		})
	}
}
