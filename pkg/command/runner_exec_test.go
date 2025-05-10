package command

import (
	"reflect"
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
