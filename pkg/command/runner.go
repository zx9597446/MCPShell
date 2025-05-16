package command

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

type RunnerType string

const (
	RunnerTypeExec        RunnerType = "exec"
	RunnerTypeSandboxExec RunnerType = "sandbox-exec"
	RunnerTypeFirejail    RunnerType = "firejail"
	RunnerTypeDocker      RunnerType = "docker"
)

// RunnerOptions is a map of options for the runner
type RunnerOptions map[string]interface{}

func (ro RunnerOptions) ToJSON() (string, error) {
	json, err := json.Marshal(ro)
	return string(json), err
}

// Runner is an interface for running commands
type Runner interface {
	Run(ctx context.Context, shell string, command string, env []string, params map[string]interface{}, tmpfile bool) (string, error)
}

// NewRunner creates a new Runner based on the given type
func NewRunner(runnerType RunnerType, options RunnerOptions, logger *log.Logger) (Runner, error) {
	switch runnerType {
	case RunnerTypeExec:
		return NewRunnerExec(options, logger)
	case RunnerTypeSandboxExec:
		return NewRunnerSandboxExec(options, logger)
	case RunnerTypeFirejail:
		return NewRunnerFirejail(options, logger)
	case RunnerTypeDocker:
		return NewDockerRunner(options, logger)
	}

	return nil, fmt.Errorf("unknown runner type: %s", runnerType)
}
