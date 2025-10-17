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
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/inercia/MCPShell/pkg/common"
)

//go:embed runner_sandbox_profile.tpl
var sandboxProfileTemplate string

// RunnerSandboxExec implements the Runner interface using macOS sandbox-exec
type RunnerSandboxExec struct {
	logger     *log.Logger
	profileTpl *template.Template
	options    RunnerSandboxExecOptions
}

// RunnerSandboxExecOptions is the options for the RunnerSandboxExec
type RunnerSandboxExecOptions struct {
	Shell             string   `json:"shell"`
	AllowNetworking   bool     `json:"allow_networking"`
	AllowUserFolders  bool     `json:"allow_user_folders"`
	AllowReadFolders  []string `json:"allow_read_folders"`
	AllowWriteFolders []string `json:"allow_write_folders"`
	AllowReadFiles    []string `json:"allow_read_files"`
	AllowWriteFiles   []string `json:"allow_write_files"`
	CustomProfile     string   `json:"custom_profile"`
}

// NewRunnerSandboxExecOptions creates a new RunnerSandboxExecOptions from a RunnerOptions
func NewRunnerSandboxExecOptions(options RunnerOptions) (RunnerSandboxExecOptions, error) {
	var reopts RunnerSandboxExecOptions
	opts, err := options.ToJSON()
	if err != nil {
		return RunnerSandboxExecOptions{}, err
	}
	err = json.Unmarshal([]byte(opts), &reopts)
	return reopts, err
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// NewRunnerSandboxExec creates a new RunnerSandboxExec with the provided logger
// If logger is nil, a default logger is created
func NewRunnerSandboxExec(options RunnerOptions, logger *log.Logger) (*RunnerSandboxExec, error) {
	if logger == nil {
		logger = log.New(os.Stderr, "runner-sandbox-exec: ", log.LstdFlags)
	}

	// Parse the sandbox profile template
	profileTpl, err := template.New("sandbox-profile").Parse(sandboxProfileTemplate)
	if err != nil {
		logger.Printf("Failed to parse sandbox profile template: %v", err)
		return nil, err
	}

	// Parse sandbox-specific options
	sandboxOpts, err := NewRunnerSandboxExecOptions(options)
	if err != nil {
		logger.Printf("Failed to parse sandbox options: %v", err)
		return nil, fmt.Errorf("failed to parse sandbox options: %w", err)
	}

	return &RunnerSandboxExec{
		logger:     logger,
		profileTpl: profileTpl,
		options:    sandboxOpts,
	}, nil
}

// Run executes a command inside the macOS sandbox and returns the output
// It implements the Runner interface
//
// note: tmpfile is ignored for sandbox because it's not supported
func (r *RunnerSandboxExec) Run(ctx context.Context, shell string, command string, env []string, params map[string]interface{}, tmpfile bool) (string, error) {
	fullCmd := command

	// Check if context is done
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue execution
	}

	// replace template variables in allow read and write folders and files
	if len(r.options.AllowReadFolders) > 0 {
		r.options.AllowReadFolders = common.ProcessTemplateListFlexible(r.options.AllowReadFolders, params)
	}
	if len(r.options.AllowWriteFolders) > 0 {
		r.options.AllowWriteFolders = common.ProcessTemplateListFlexible(r.options.AllowWriteFolders, params)
	}
	if len(r.options.AllowReadFiles) > 0 {
		r.options.AllowReadFiles = common.ProcessTemplateListFlexible(r.options.AllowReadFiles, params)

		// For macOS sandbox, we need to allow read access to parent directories
		// of files to enable directory traversal
		for _, filePath := range r.options.AllowReadFiles {
			dir := filepath.Dir(filePath)
			// Add parent directory if not already in the list
			if !contains(r.options.AllowReadFolders, dir) {
				r.options.AllowReadFolders = append(r.options.AllowReadFolders, dir)
				r.logger.Printf("[DEBUG] Added parent directory to allow list: %s", dir)
			}
		}
	}
	if len(r.options.AllowWriteFiles) > 0 {
		r.options.AllowWriteFiles = common.ProcessTemplateListFlexible(r.options.AllowWriteFiles, params)

		// For macOS sandbox, we need to allow write access to parent directories
		// of files to enable directory traversal
		for _, filePath := range r.options.AllowWriteFiles {
			dir := filepath.Dir(filePath)
			// Add parent directory if not already in the list
			if !contains(r.options.AllowWriteFolders, dir) {
				r.options.AllowWriteFolders = append(r.options.AllowWriteFolders, dir)
				r.logger.Printf("[DEBUG] Added parent directory to allow list: %s", dir)
			}
		}
	}

	// Generate the profile by rendering the template
	var profileBuf bytes.Buffer
	if err := r.profileTpl.Execute(&profileBuf, r.options); err != nil {
		r.logger.Printf("Failed to render sandbox profile template: %v", err)
		return "", fmt.Errorf("failed to render sandbox profile: %w", err)
	}

	profile := profileBuf.String()
	r.logger.Printf("Sandbox options: %+v", r.options)
	r.logger.Printf("Generated sandbox profile:\n%s", profile)

	// Create a temporary file for the sandbox profile
	profileFile, err := os.CreateTemp("", "sandbox-profile-*.sb")
	if err != nil {
		r.logger.Printf("Failed to create temporary profile file: %v", err)
		return "", fmt.Errorf("failed to create temporary profile file: %w", err)
	}
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

	var execCmd *exec.Cmd

	// Check if we can optimize by running a single executable directly
	if isSingleExecutableCommand(fullCmd) {
		r.logger.Printf("Optimization: running single executable command directly: %s", fullCmd)
		execCmd = exec.CommandContext(ctx, "sandbox-exec", "-f", profileFile.Name(), fullCmd)
	} else {
		// Create a temporary file for the command
		tmpScript, err := os.CreateTemp("", "sandbox-script-*.sh")
		if err != nil {
			r.logger.Printf("Failed to create temporary command file: %v", err)
			return "", fmt.Errorf("failed to create temporary command file: %w", err)
		}
		// Ensure temporary file is deleted when this function exits
		defer func() {
			tmpScriptPath := tmpScript.Name()
			if err := tmpScript.Close(); err != nil {
				r.logger.Printf("Warning: failed to close script file: %v", err)
			}
			if err := os.Remove(tmpScriptPath); err != nil {
				r.logger.Printf("Warning: failed to remove temporary script file: %v", err)
			}
		}()

		// Write the command to the temporary file
		if _, err := tmpScript.WriteString(fullCmd); err != nil {
			r.logger.Printf("Failed to write command to temporary file: %v", err)
			return "", fmt.Errorf("failed to write command to temporary file: %w", err)
		}

		// Flush data to ensure it's written to disk
		if err := tmpScript.Sync(); err != nil {
			r.logger.Printf("Failed to sync script file: %v", err)
			return "", fmt.Errorf("failed to sync script file: %w", err)
		}

		// Make the temporary file executable
		if err := os.Chmod(tmpScript.Name(), 0o700); err != nil {
			r.logger.Printf("Failed to make temporary file executable: %v", err)
			return "", fmt.Errorf("failed to make temporary file executable: %w", err)
		}

		execCmd = exec.CommandContext(ctx, "sandbox-exec", "-f", profileFile.Name(), tmpScript.Name())
	}

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

// CheckImplicitRequirements checks if the runner meets its implicit requirements
// SandboxExec runner requires macOS and the sandbox-exec executable
func (r *RunnerSandboxExec) CheckImplicitRequirements() error {
	// Sandbox exec is macOS only
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("sandbox-exec runner requires macOS")
	}

	// Check if sandbox-exec is available
	if !common.CheckExecutableExists("sandbox-exec") {
		return fmt.Errorf("sandbox-exec executable not found in PATH")
	}

	return nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
