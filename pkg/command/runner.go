package command

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inercia/MCPShell/pkg/common"
)

// RunnerType is an identifier for the type of runner to use.
// Each runner has its own set of implicit requirements that are checked
// automatically, so users don't need to explicitly specify common requirements
// in their tool configurations.
type RunnerType string

const (
	// RunnerTypeExec is the standard command execution runner with no additional requirements
	RunnerTypeExec RunnerType = "exec"

	// RunnerTypeSandboxExec is the macOS-specific sandbox-exec runner
	// Implicit requirements: OS=darwin, executables=[sandbox-exec]
	RunnerTypeSandboxExec RunnerType = "sandbox-exec"

	// RunnerTypeFirejail is the Linux-specific firejail runner
	// Implicit requirements: OS=linux, executables=[firejail]
	RunnerTypeFirejail RunnerType = "firejail"

	// RunnerTypeDocker is the Docker-based runner
	// Implicit requirements: executables=[docker]
	RunnerTypeDocker RunnerType = "docker"
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
	CheckImplicitRequirements() error
}

// NewRunner creates a new Runner based on the given type
func NewRunner(runnerType RunnerType, options RunnerOptions, logger *common.Logger) (Runner, error) {
	var runner Runner
	var err error

	// Create the runner instance based on type
	switch runnerType {
	case RunnerTypeExec:
		runner, err = NewRunnerExec(options, logger)
	case RunnerTypeSandboxExec:
		runner, err = NewRunnerSandboxExec(options, logger)
	case RunnerTypeFirejail:
		runner, err = NewRunnerFirejail(options, logger)
	case RunnerTypeDocker:
		runner, err = NewDockerRunner(options, logger)
	default:
		return nil, fmt.Errorf("unknown runner type: %s", runnerType)
	}

	// Check if runner creation failed
	if err != nil {
		return nil, err
	}

	// Check implicit requirements for the created runner
	if err := runner.CheckImplicitRequirements(); err != nil {
		if logger != nil {
			logger.Debug("Runner %s failed implicit requirements check: %v", runnerType, err)
		}
		return nil, err
	}

	return runner, nil
}
