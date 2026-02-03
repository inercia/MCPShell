// Package runner provides isolated command execution environments.
//
// This package defines the Runner interface and implementations for executing
// commands in various isolation environments including direct execution,
// firejail (Linux), sandbox-exec (macOS), and Docker containers.
package runner

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inercia/MCPShell/pkg/common"
)

// Type is an identifier for the type of runner to use.
// Each runner has its own set of implicit requirements that are checked
// automatically, so users don't need to explicitly specify common requirements
// in their tool configurations.
type Type string

const (
	// TypeExec is the standard command execution runner with no additional requirements
	TypeExec Type = "exec"

	// TypeSandboxExec is the macOS-specific sandbox-exec runner
	// Implicit requirements: OS=darwin, executables=[sandbox-exec]
	TypeSandboxExec Type = "sandbox-exec"

	// TypeFirejail is the Linux-specific firejail runner
	// Implicit requirements: OS=linux, executables=[firejail]
	TypeFirejail Type = "firejail"

	// TypeDocker is the Docker-based runner
	// Implicit requirements: executables=[docker]
	TypeDocker Type = "docker"
)

// Options is a map of options for the runner
type Options map[string]interface{}

// ToJSON converts the options to a JSON string
func (o Options) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(o)
	return string(jsonBytes), err
}

// Runner is an interface for running commands in isolated environments
type Runner interface {
	// Run executes a command and returns the output.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - shell: The shell to use for execution (empty for default)
	//   - command: The command to execute
	//   - env: Environment variables in KEY=VALUE format
	//   - params: Template parameters for variable substitution
	//   - tmpfile: Whether to use a temporary file for the command
	//
	// Returns:
	//   - The command output as a string
	//   - An error if execution fails
	Run(ctx context.Context, shell string, command string, env []string, params map[string]interface{}, tmpfile bool) (string, error)

	// CheckImplicitRequirements verifies that the runner's prerequisites are met.
	// This includes checking for required executables, OS compatibility, etc.
	//
	// Returns:
	//   - nil if all requirements are satisfied
	//   - An error describing which requirement is not met
	CheckImplicitRequirements() error
}

// New creates a new Runner based on the given type.
//
// Parameters:
//   - runnerType: The type of runner to create
//   - options: Configuration options for the runner
//   - logger: Logger for debug output (uses global logger if nil)
//
// Returns:
//   - A Runner instance if successful
//   - An error if creation fails or requirements are not met
func New(runnerType Type, options Options, logger *common.Logger) (Runner, error) {
	var runner Runner
	var err error

	// Create the runner instance based on type
	switch runnerType {
	case TypeExec:
		runner, err = NewExec(options, logger)
	case TypeSandboxExec:
		runner, err = NewSandboxExec(options, logger)
	case TypeFirejail:
		runner, err = NewFirejail(options, logger)
	case TypeDocker:
		runner, err = NewDocker(options, logger)
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
