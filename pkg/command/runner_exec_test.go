package command

import (
	"context"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestNewRunnerExecOptions(t *testing.T) {
	tests := []struct {
		name    string
		options RunnerOptions
		want    RunnerExecOptions
		wantErr bool
	}{
		{
			name: "valid options with shell",
			options: RunnerOptions{
				"shell": "/bin/bash",
			},
			want: RunnerExecOptions{
				Shell: "/bin/bash",
			},
			wantErr: false,
		},
		{
			name:    "empty options",
			options: RunnerOptions{},
			want:    RunnerExecOptions{},
			wantErr: false,
		},
		{
			name: "options with additional fields",
			options: RunnerOptions{
				"shell": "/bin/zsh",
				"extra": "value",
			},
			want: RunnerExecOptions{
				Shell: "/bin/zsh",
			},
			wantErr: false,
		},
		{
			name: "options with numeric shell as string",
			options: RunnerOptions{
				"shell": "123",
			},
			want: RunnerExecOptions{
				Shell: "123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRunnerExecOptions(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRunnerExecOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRunnerExecOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunnerExec_Run(t *testing.T) {
	tests := []struct {
		name    string
		shell   string
		command string
		args    []string
		env     []string
		params  map[string]interface{}
		want    string
		wantErr bool
	}{
		{
			name:    "simple echo command",
			shell:   "",
			command: "echo",
			args:    []string{"hello", "world"},
			env:     nil,
			params:  nil,
			want:    "hello world",
			wantErr: false,
		},
		{
			name:    "command with special characters",
			shell:   "",
			command: "echo",
			args:    []string{"hello", "world", "with", "special", "chars:", "$PATH", ">&", "!"},
			env:     nil,
			params:  nil,
			want:    "hello world with special chars: $PATH >& !",
			wantErr: false,
		},
		{
			name:    "command with environment variable",
			shell:   "",
			command: "echo",
			args:    []string{"$TEST_VAR"},
			env:     []string{"TEST_VAR=test_value"},
			params:  nil,
			want:    "$TEST_VAR",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.New(os.Stderr, "test-runner-exec: ", log.LstdFlags)
			r, err := NewRunnerExec(RunnerOptions{}, logger)
			if err != nil {
				t.Fatalf("Failed to create RunnerExec: %v", err)
			}

			got, err := r.Run(context.Background(), tt.shell, tt.command, tt.args, tt.env, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunnerExec.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Trim any trailing newlines for comparison
			got = strings.TrimSpace(got)

			if got != tt.want {
				t.Errorf("RunnerExec.Run() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunnerExec_RunWithEnvExpansion(t *testing.T) {
	// This test demonstrates using the -c flag to execute a command with environment variable expansion
	logger := log.New(os.Stderr, "test-runner-exec-env: ", log.LstdFlags)

	r, err := NewRunnerExec(RunnerOptions{}, logger)
	if err != nil {
		t.Fatalf("Failed to create RunnerExec: %v", err)
	}

	// Use the shell's -c flag directly to execute a command that expands an environment variable
	output, err := r.Run(
		context.Background(),
		"",
		"/bin/sh",
		[]string{"-c", "echo $TEST_VAR"},
		[]string{"TEST_VAR=test_value_expanded"},
		nil,
	)

	if err != nil {
		t.Fatalf("RunnerExec.Run() error = %v", err)
	}

	output = strings.TrimSpace(output)
	expected := "test_value_expanded"

	if output != expected {
		t.Errorf("Environment variable expansion failed: got %q, want %q", output, expected)
	}
}
