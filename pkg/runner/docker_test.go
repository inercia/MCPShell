package runner

import (
	"context"
	"os/exec"
	"runtime"
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

func TestDockerInitialization(t *testing.T) {
	// Skip on Windows - Alpine Linux doesn't support Windows containers
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Docker test on Windows - Alpine Linux image not compatible with Windows containers")
	}

	if !checkDockerRunning() {
		t.Skip("Docker not installed or not running, skipping test")
	}

	logger, _ := common.NewLogger("test-docker: ", "", common.LogLevelInfo, false)

	testCases := []struct {
		name        string
		options     Options
		expectError bool
	}{
		{
			name: "Valid options",
			options: Options{
				"image": "alpine:latest",
			},
			expectError: false,
		},
		{
			name:        "Missing image",
			options:     Options{},
			expectError: true,
		},
		{
			name: "Full options",
			options: Options{
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
			_, err := NewDocker(tc.options, logger)
			if tc.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestDocker_Run_Basic(t *testing.T) {
	// Skip on Windows - Alpine Linux doesn't support Windows containers
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Docker test on Windows - Alpine Linux image not compatible with Windows containers")
	}

	// Skip if docker is not available or not running
	if !checkDockerRunning() {
		t.Skip("Docker not installed or not running, skipping test")
	}

	logger, _ := common.NewLogger("test-docker: ", "", common.LogLevelInfo, false)

	// Create a runner with alpine image
	r, err := NewDocker(Options{
		"image": "alpine:latest",
	}, logger)

	if err != nil {
		t.Fatalf("Failed to create Docker runner: %v", err)
	}

	// Test a simple echo command
	output, err := r.Run(context.Background(), "", "echo 'Hello from Docker'", nil, nil, false)
	if err != nil {
		t.Errorf("Failed to run command: %v", err)
	}

	// Check the output
	expected := "Hello from Docker"
	if output != expected {
		t.Errorf("Expected output %q, got %q", expected, output)
	}
}

func TestDocker_Run_EnvironmentVariables(t *testing.T) {
	// Skip on Windows - Alpine Linux doesn't support Windows containers
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Docker test on Windows - Alpine Linux image not compatible with Windows containers")
	}

	// Skip if docker is not available or not running
	if !checkDockerRunning() {
		t.Skip("Docker not installed or not running, skipping test")
	}

	logger, _ := common.NewLogger("test-docker: ", "", common.LogLevelInfo, false)

	// Create a runner with alpine image
	r, err := NewDocker(Options{
		"image": "alpine:latest",
	}, logger)

	if err != nil {
		t.Fatalf("Failed to create Docker runner: %v", err)
	}

	// Define environment variables to pass to the container
	env := []string{
		"TEST_VAR1=test_value1",
		"TEST_VAR2=test_value2",
	}

	// Run a command that echoes the environment variables
	output, err := r.Run(context.Background(), "", "echo $TEST_VAR1,$TEST_VAR2", env, nil, false)
	if err != nil {
		t.Errorf("Failed to run command with environment variables: %v", err)
	}

	// Check the output contains the environment variable values
	expected := "test_value1,test_value2"
	if output != expected {
		t.Errorf("Environment variables not correctly passed. Expected %q, got %q", expected, output)
	}
}

func TestDocker_Optimization_SingleExecutable(t *testing.T) {
	// Skip on Windows - Alpine Linux doesn't support Windows containers
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Docker test on Windows - Alpine Linux image not compatible with Windows containers")
	}

	if !checkDockerRunning() {
		t.Skip("Docker not installed or not running, skipping test")
	}
	logger, _ := common.NewLogger("test-docker-opt: ", "", common.LogLevelInfo, false)
	r, err := NewDocker(Options{
		"image": "alpine:latest",
	}, logger)
	if err != nil {
		t.Fatalf("Failed to create Docker runner: %v", err)
	}
	// Should succeed: /bin/ls is a single executable in alpine
	output, err := r.Run(context.Background(), "", "/bin/ls", nil, nil, false)
	if err != nil {
		t.Errorf("Expected /bin/ls to run without error in Docker, got: %v", err)
	}
	if len(output) == 0 {
		t.Errorf("Expected output from /bin/ls in Docker, got empty string")
	}
	// Should NOT optimize: command with arguments
	_, err2 := r.Run(context.Background(), "", "/bin/ls -l", nil, nil, false)
	if err2 != nil && !strings.Contains(err2.Error(), "no such file") {
		t.Logf("Expected failure for /bin/ls -l as a single executable in Docker: %v", err2)
	}
}

func TestNewDockerOptions(t *testing.T) {
	testCases := []struct {
		name        string
		input       Options
		expected    DockerOptions
		expectError bool
	}{
		{
			name: "minimal valid options",
			input: Options{
				"image": "alpine:latest",
			},
			expected: DockerOptions{
				Image:            "alpine:latest",
				AllowNetworking:  true,
				MemorySwappiness: -1,
			},
			expectError: false,
		},
		{
			name:        "missing required image",
			input:       Options{},
			expected:    DockerOptions{},
			expectError: true,
		},
		{
			name: "comprehensive options",
			input: Options{
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
			expected: DockerOptions{
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
			result, err := NewDockerOptions(tc.input)

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
			if result.AllowNetworking != tc.expected.AllowNetworking {
				t.Errorf("AllowNetworking: expected %v, got %v", tc.expected.AllowNetworking, result.AllowNetworking)
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
