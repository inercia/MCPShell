package command

import (
	"context"
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

	logger, _ := common.NewLogger("test-docker: ", "", common.LogLevelInfo, false)

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

	logger, _ := common.NewLogger("test-docker: ", "", common.LogLevelInfo, false)

	// Create a runner with alpine image
	runner, err := NewDockerRunner(RunnerOptions{
		"image": "alpine:latest",
	}, logger)

	if err != nil {
		t.Fatalf("Failed to create Docker runner: %v", err)
	}

	// Test a simple echo command (this should work even in GitHub Actions)
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

	// Check if running in GitHub Actions
	inGitHubActions := os.Getenv("GITHUB_ACTIONS") == "true"
	if inGitHubActions {
		t.Skip("Skipping network test in GitHub Actions environment")
	}

	logger, _ := common.NewLogger("test-docker: ", "", common.LogLevelInfo, false)

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
			// Skip network-enabled tests in GitHub Actions

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

	logger, _ := common.NewLogger("test-docker: ", "", common.LogLevelInfo, false)

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

	logger, _ := common.NewLogger("test-docker: ", "", common.LogLevelInfo, false)

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

func TestDockerRunner_Optimization_SingleExecutable(t *testing.T) {
	if !checkDockerRunning() {
		t.Skip("Docker not installed or not running, skipping test")
	}
	logger, _ := common.NewLogger("test-docker-opt: ", "", common.LogLevelInfo, false)
	runner, err := NewDockerRunner(RunnerOptions{
		"image": "alpine:latest",
	}, logger)
	if err != nil {
		t.Fatalf("Failed to create Docker runner: %v", err)
	}
	// Should succeed: /bin/ls is a single executable in alpine
	output, err := runner.Run(context.Background(), "", "/bin/ls", nil, nil, false)
	if err != nil {
		t.Errorf("Expected /bin/ls to run without error in Docker, got: %v", err)
	}
	if len(output) == 0 {
		t.Errorf("Expected output from /bin/ls in Docker, got empty string")
	}
	// Should NOT optimize: command with arguments
	_, err2 := runner.Run(context.Background(), "", "/bin/ls -l", nil, nil, false)
	if err2 != nil && !strings.Contains(err2.Error(), "no such file") {
		t.Logf("Expected failure for /bin/ls -l as a single executable in Docker: %v", err2)
	}
}

func TestNewDockerRunnerOptions(t *testing.T) {
	testCases := []struct {
		name        string
		input       RunnerOptions
		expected    DockerRunnerOptions
		expectError bool
	}{
		{
			name: "minimal valid options",
			input: RunnerOptions{
				"image": "alpine:latest",
			},
			expected: DockerRunnerOptions{
				Image:            "alpine:latest",
				AllowNetworking:  true,
				MemorySwappiness: -1,
			},
			expectError: false,
		},
		{
			name:        "missing required image",
			input:       RunnerOptions{},
			expected:    DockerRunnerOptions{},
			expectError: true,
		},
		{
			name: "comprehensive options",
			input: RunnerOptions{
				"image":              "ubuntu:20.04",
				"docker_run_opts":    "--cpus 2",
				"mounts":             []interface{}{"/host:/container", "/tmp:/tmp"},
				"allow_networking":   false,
				"network":            "host",
				"user":               "nobody",
				"workdir":            "/app",
				"prepare_command":    "apt-get update",
				"memory":             "512m",
				"memory_reservation": "256m",
				"memory_swap":        "1g",
				"memory_swappiness":  float64(10),
				"cap_add":            []interface{}{"SYS_ADMIN"},
				"cap_drop":           []interface{}{"NET_ADMIN"},
				"dns":                []interface{}{"8.8.8.8"},
				"dns_search":         []interface{}{"example.com"},
				"platform":           "linux/amd64",
			},
			expected: DockerRunnerOptions{
				Image:             "ubuntu:20.04",
				DockerRunOpts:     "--cpus 2",
				Mounts:            []string{"/host:/container", "/tmp:/tmp"},
				AllowNetworking:   false,
				Network:           "host",
				User:              "nobody",
				WorkDir:           "/app",
				PrepareCommand:    "apt-get update",
				Memory:            "512m",
				MemoryReservation: "256m",
				MemorySwap:        "1g",
				MemorySwappiness:  10,
				CapAdd:            []string{"SYS_ADMIN"},
				CapDrop:           []string{"NET_ADMIN"},
				DNS:               []string{"8.8.8.8"},
				DNSSearch:         []string{"example.com"},
				Platform:          "linux/amd64",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := NewDockerRunnerOptions(tc.input)

			// Check error expectation
			if tc.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Skip further checks if we expected an error
			if tc.expectError {
				return
			}

			// Check specific fields
			if result.Image != tc.expected.Image {
				t.Errorf("Image: expected %q, got %q", tc.expected.Image, result.Image)
			}
			if result.DockerRunOpts != tc.expected.DockerRunOpts {
				t.Errorf("DockerRunOpts: expected %q, got %q", tc.expected.DockerRunOpts, result.DockerRunOpts)
			}
			if result.AllowNetworking != tc.expected.AllowNetworking {
				t.Errorf("AllowNetworking: expected %v, got %v", tc.expected.AllowNetworking, result.AllowNetworking)
			}
			if result.Network != tc.expected.Network {
				t.Errorf("Network: expected %q, got %q", tc.expected.Network, result.Network)
			}
			if result.User != tc.expected.User {
				t.Errorf("User: expected %q, got %q", tc.expected.User, result.User)
			}
			if result.WorkDir != tc.expected.WorkDir {
				t.Errorf("WorkDir: expected %q, got %q", tc.expected.WorkDir, result.WorkDir)
			}
			if result.PrepareCommand != tc.expected.PrepareCommand {
				t.Errorf("PrepareCommand: expected %q, got %q", tc.expected.PrepareCommand, result.PrepareCommand)
			}

			// Check slice fields
			if !compareStringSlices(result.Mounts, tc.expected.Mounts) {
				t.Errorf("Mounts: expected %v, got %v", tc.expected.Mounts, result.Mounts)
			}
			if !compareStringSlices(result.CapAdd, tc.expected.CapAdd) {
				t.Errorf("CapAdd: expected %v, got %v", tc.expected.CapAdd, result.CapAdd)
			}
			if !compareStringSlices(result.CapDrop, tc.expected.CapDrop) {
				t.Errorf("CapDrop: expected %v, got %v", tc.expected.CapDrop, result.CapDrop)
			}
			if !compareStringSlices(result.DNS, tc.expected.DNS) {
				t.Errorf("DNS: expected %v, got %v", tc.expected.DNS, result.DNS)
			}
			if !compareStringSlices(result.DNSSearch, tc.expected.DNSSearch) {
				t.Errorf("DNSSearch: expected %v, got %v", tc.expected.DNSSearch, result.DNSSearch)
			}
		})
	}
}

// Helper function to compare string slices
func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
