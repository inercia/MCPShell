// Package command provides functions for creating and executing command handlers.
package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/inercia/MCPShell/pkg/common"
)

// DockerRunner executes commands inside a Docker container.
type DockerRunner struct {
	logger *common.Logger
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

	// Specific network to connect container to (e.g. "host", "bridge", or custom network name)
	Network string `json:"network"`

	// User to run as inside the container (defaults to current user)
	User string `json:"user"`

	// Working directory inside the container
	WorkDir string `json:"workdir"`

	// PrepareCommand is a command to run before the main command
	PrepareCommand string `json:"prepare_command"`

	// Memory limit (e.g. "512m", "1g")
	Memory string `json:"memory"`

	// Memory soft limit (e.g. "256m", "512m")
	MemoryReservation string `json:"memory_reservation"`

	// Swap limit equal to memory plus swap: '-1' to enable unlimited swap
	MemorySwap string `json:"memory_swap"`

	// Tune container memory swappiness (0 to 100)
	MemorySwappiness int `json:"memory_swappiness"`

	// Linux capabilities to add to the container
	CapAdd []string `json:"cap_add"`

	// Linux capabilities to drop from the container
	CapDrop []string `json:"cap_drop"`

	// Custom DNS servers for the container
	DNS []string `json:"dns"`

	// Custom DNS search domains for the container
	DNSSearch []string `json:"dns_search"`

	// Set platform if server is multi-platform capable (e.g., "linux/amd64", "linux/arm64")
	Platform string `json:"platform"`
}

// GetBaseDockerCommand creates the common parts of a docker run command with all configured options.
// It returns a slice of command parts that can be further customized by the calling method.
func (o *DockerRunnerOptions) GetBaseDockerCommand(env []string) []string {
	// Start with basic docker run command
	parts := []string{"docker run --rm"}

	// Add networking option
	if !o.AllowNetworking {
		parts = append(parts, "--network none")
	} else if o.Network != "" {
		parts = append(parts, fmt.Sprintf("--network %s", o.Network))
	}

	// Add user if specified
	if o.User != "" {
		parts = append(parts, fmt.Sprintf("--user %s", o.User))
	}

	// Add working directory if specified
	if o.WorkDir != "" {
		parts = append(parts, fmt.Sprintf("--workdir %s", o.WorkDir))
	}

	// Add memory options if specified
	if o.Memory != "" {
		parts = append(parts, fmt.Sprintf("--memory %s", o.Memory))
	}

	if o.MemoryReservation != "" {
		parts = append(parts, fmt.Sprintf("--memory-reservation %s", o.MemoryReservation))
	}

	if o.MemorySwap != "" {
		parts = append(parts, fmt.Sprintf("--memory-swap %s", o.MemorySwap))
	}

	if o.MemorySwappiness != -1 {
		parts = append(parts, fmt.Sprintf("--memory-swappiness %d", o.MemorySwappiness))
	}

	// Add Linux capabilities options
	for _, cap := range o.CapAdd {
		parts = append(parts, fmt.Sprintf("--cap-add %s", cap))
	}

	for _, cap := range o.CapDrop {
		parts = append(parts, fmt.Sprintf("--cap-drop %s", cap))
	}

	// Add DNS servers
	for _, dns := range o.DNS {
		parts = append(parts, fmt.Sprintf("--dns %s", dns))
	}

	// Add DNS search domains
	for _, dnsSearch := range o.DNSSearch {
		parts = append(parts, fmt.Sprintf("--dns-search %s", dnsSearch))
	}

	// Add platform if specified
	if o.Platform != "" {
		parts = append(parts, fmt.Sprintf("--platform %s", o.Platform))
	}

	// Add custom docker run options
	if o.DockerRunOpts != "" {
		parts = append(parts, o.DockerRunOpts)
	}

	// Add additional mounts
	for _, mount := range o.Mounts {
		parts = append(parts, fmt.Sprintf("-v %s", mount))
	}

	// Add environment variables
	for _, e := range env {
		parts = append(parts, fmt.Sprintf("-e %s", e))
	}

	return parts
}

// GetDockerCommand constructs the docker run command with a script file.
func (o *DockerRunnerOptions) GetDockerCommand(scriptFile string, env []string) string {
	// Get base docker command parts
	parts := o.GetBaseDockerCommand(env)

	// Mount the script file
	scriptName := filepath.Base(scriptFile)
	containerScriptPath := filepath.Join("/tmp", scriptName)
	parts = append(parts, fmt.Sprintf("-v %s:%s", scriptFile, containerScriptPath))

	// Add image and the command to execute the script
	parts = append(parts, o.Image)
	parts = append(parts, fmt.Sprintf("sh %s", containerScriptPath))

	// Join all parts
	return strings.Join(parts, " ")
}

// GetDirectExecutionCommand constructs the docker run command for direct executable execution.
// This is used to optimize the case where we're just running a single executable without a temp script.
func (o *DockerRunnerOptions) GetDirectExecutionCommand(cmd string, env []string) string {
	// Get base docker command parts
	parts := o.GetBaseDockerCommand(env)

	// Add image and direct command
	parts = append(parts, o.Image)
	parts = append(parts, cmd)

	// Join all parts into a single command
	return strings.Join(parts, " ")
}

// NewDockerRunnerOptions extracts Docker-specific options from generic runner options.
func NewDockerRunnerOptions(genericOpts RunnerOptions) (DockerRunnerOptions, error) {
	opts := DockerRunnerOptions{
		AllowNetworking:  true, // Default to allowing networking
		User:             "",   // Default to Docker's default user
		WorkDir:          "",   // Default to Docker's default working directory
		MemorySwappiness: -1,   // Default to Docker's default swappiness
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

	// Parse network option
	if network, ok := genericOpts["network"].(string); ok {
		opts.Network = network
	}

	// Parse user option
	if user, ok := genericOpts["user"].(string); ok {
		opts.User = user
	}

	// Parse working directory option
	if workDir, ok := genericOpts["workdir"].(string); ok {
		opts.WorkDir = workDir
	}

	// Parse prepare command option
	if prepareCommand, ok := genericOpts["prepare_command"].(string); ok {
		opts.PrepareCommand = prepareCommand
	}

	// Parse memory option
	if memory, ok := genericOpts["memory"].(string); ok {
		opts.Memory = memory
	}

	// Parse memory reservation option
	if memoryReservation, ok := genericOpts["memory_reservation"].(string); ok {
		opts.MemoryReservation = memoryReservation
	}

	// Parse memory swap option
	if memorySwap, ok := genericOpts["memory_swap"].(string); ok {
		opts.MemorySwap = memorySwap
	}

	// Parse memory swappiness option (integer)
	if swappiness, ok := genericOpts["memory_swappiness"].(float64); ok {
		opts.MemorySwappiness = int(swappiness)
	}

	// Parse capabilities to add
	if capAdd, ok := genericOpts["cap_add"].([]interface{}); ok {
		for _, cap := range capAdd {
			if capStr, ok := cap.(string); ok {
				opts.CapAdd = append(opts.CapAdd, capStr)
			}
		}
	}

	// Parse capabilities to drop
	if capDrop, ok := genericOpts["cap_drop"].([]interface{}); ok {
		for _, cap := range capDrop {
			if capStr, ok := cap.(string); ok {
				opts.CapDrop = append(opts.CapDrop, capStr)
			}
		}
	}

	// Parse DNS servers
	if dns, ok := genericOpts["dns"].([]interface{}); ok {
		for _, server := range dns {
			if serverStr, ok := server.(string); ok {
				opts.DNS = append(opts.DNS, serverStr)
			}
		}
	}

	// Parse DNS search domains
	if dnsSearch, ok := genericOpts["dns_search"].([]interface{}); ok {
		for _, domain := range dnsSearch {
			if domainStr, ok := domain.(string); ok {
				opts.DNSSearch = append(opts.DNSSearch, domainStr)
			}
		}
	}

	// Parse platform option
	if platform, ok := genericOpts["platform"].(string); ok {
		opts.Platform = platform
	}

	return opts, nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// NewDockerRunner creates a new Docker runner with the specified options.
func NewDockerRunner(options RunnerOptions, logger *common.Logger) (*DockerRunner, error) {
	if logger == nil {
		logger = common.GetLogger()
	}

	dockerOpts, err := NewDockerRunnerOptions(options)
	if err != nil {
		return nil, err
	}

	// Docker executable and daemon checks are now handled by CheckImplicitRequirements()
	return &DockerRunner{
		logger: logger,
		opts:   dockerOpts,
	}, nil
}

// CheckImplicitRequirements checks if the runner meets its implicit requirements
// Docker runner requires the docker executable and a running daemon
func (r *DockerRunner) CheckImplicitRequirements() error {
	// Check if docker executable exists
	if !common.CheckExecutableExists("docker") {
		return fmt.Errorf("docker executable not found in PATH")
	}

	// Check if Docker daemon is running
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "stats", "--no-stream")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker daemon is not running: %w", err)
	}

	return nil
}

// Run executes the command using Docker.
func (r *DockerRunner) Run(ctx context.Context, shell string, cmd string, env []string, params map[string]interface{}, tmpfile bool) (string, error) {
	// Create an exec runner that we'll use to execute the docker command
	execRunner, err := NewRunnerExec(RunnerOptions{}, r.logger)
	if err != nil {
		return "", fmt.Errorf("failed to create exec runner: %w", err)
	}

	var dockerCmd string

	// Determine if we should run directly or via script
	if isSingleExecutableCommand(cmd) {
		r.logger.Debug("Optimization: running single executable command directly in Docker: %s", cmd)

		// Build docker command to directly execute the command without a temp script
		dockerCmd = r.opts.GetDirectExecutionCommand(cmd, env)
	} else {
		// Create a temporary script file
		scriptFile, err := r.createScriptFile(shell, cmd, env)
		if err != nil {
			return "", fmt.Errorf("failed to create script file: %w", err)
		}

		// Clean up the temporary script file when done
		defer func() {
			if err := os.Remove(scriptFile); err != nil {
				r.logger.Debug("Warning: failed to remove temporary script file %s: %v", scriptFile, err)
			}
		}()

		r.logger.Debug("Created temporary script file: %s", scriptFile)

		// Construct the docker run command with the script file
		dockerCmd = r.opts.GetDockerCommand(scriptFile, env)
	}

	r.logger.Debug("Running command in Docker: %s", dockerCmd)

	// Run the docker command - we set tmpfile to false because dockerCmd is already a full command
	output, err := execRunner.Run(ctx, "sh", dockerCmd, nil, params, false)
	if err != nil {
		return "", fmt.Errorf("docker command execution failed: %w", err)
	}

	return output, nil
}

// createScriptFile writes the command to a temporary script file.
func (r *DockerRunner) createScriptFile(shell string, cmd string, env []string) (string, error) {
	// Create a temporary file with a specific pattern
	tmpFile, err := os.CreateTemp("", "mcpshell-docker-*.sh")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary script file: %w", err)
	}

	// Get the name for later usage
	scriptPath := tmpFile.Name()

	// Prepare script content
	var content strings.Builder
	content.WriteString("#!/bin/sh\n\n")

	// Add environment variables
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			fmt.Fprintf(&content, "export %s=%s\n", parts[0], parts[1])
		}
	}

	// Add preparation command if specified
	if r.opts.PrepareCommand != "" {
		content.WriteString("\n# Preparation commands\n")
		content.WriteString(r.opts.PrepareCommand)
		content.WriteString("\n\n")
		r.logger.Debug("Added preparation command to script: %s", r.opts.PrepareCommand)
	}

	// Add the main command
	content.WriteString("# Main command to execute\n")
	if shell != "" {
		fmt.Fprintf(&content, "exec %s -c %q\n", shell, cmd)
	} else {
		fmt.Fprintf(&content, "exec sh -c %q\n", cmd)
	}

	// Write the content to the file
	if _, err := tmpFile.WriteString(content.String()); err != nil {
		// Close and remove the file in case of an error
		_ = tmpFile.Close()       // Ignore close error, we already have a write error
		_ = os.Remove(scriptPath) // Best effort cleanup
		return "", fmt.Errorf("failed to write to temporary script file: %w", err)
	}

	// Make the file executable (chmod +x)
	if err := os.Chmod(scriptPath, 0755); err != nil {
		_ = tmpFile.Close()       // Ignore close error, we already have a chmod error
		_ = os.Remove(scriptPath) // Best effort cleanup
		return "", fmt.Errorf("failed to make script file executable: %w", err)
	}

	// Close the file
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(scriptPath) // Best effort cleanup
		return "", fmt.Errorf("failed to close temporary script file: %w", err)
	}

	r.logger.Debug("Created temporary script file at: %s", scriptPath)
	return scriptPath, nil
}
