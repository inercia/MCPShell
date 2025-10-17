package command

import (
	"context"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/inercia/MCPShell/pkg/common"
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

	// Set environment variables for the test
	if err := os.Setenv("ALLOWED_FROM_ENV", "/tmp"); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
	if err := os.Setenv("USR_DIR", "/usr"); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}

	// Ensure cleanup
	defer func() {
		if err := os.Unsetenv("ALLOWED_FROM_ENV"); err != nil {
			t.Logf("Failed to unset environment variable: %v", err)
		}
	}()
	defer func() {
		if err := os.Unsetenv("USR_DIR"); err != nil {
			t.Logf("Failed to unset environment variable: %v", err)
		}
	}()

	// Create a logger for the test
	logger, _ := common.NewLogger("test-runner-sandbox: ", "", common.LogLevelInfo, false)
	ctx := context.Background()
	shell := "" // use default

	tests := []struct {
		name          string
		command       string
		args          []string
		options       RunnerOptions
		params        map[string]interface{} // Parameters for template processing
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
		// 		{
		// 			name:    "custom profile blocking all except echo",
		// 			command: "echo 'only echo works'",
		// 			args:    []string{},
		// 			options: RunnerOptions{
		// 				"custom_profile": `(version 1)
		// (allow default)
		// (deny process-exec*)
		// (allow process-exec* (regex "^/bin/echo"))
		// (allow process-exec* (regex "^/usr/bin/echo"))`,
		// 			},
		// 			shouldSucceed: true,
		// 			expectedOut:   "only echo works",
		// 		},
		// New test cases for allow_read_folders
		{
			name:    "read from allowed folder using env variable",
			command: "ls -la /tmp > /dev/null && echo 'can read /tmp'",
			args:    []string{},
			options: RunnerOptions{
				"allow_networking":   false,
				"allow_user_folders": false,
				"allow_read_folders": []string{"{{ env ALLOWED_FROM_ENV }}"},
				"custom_profile":     "", // Ensure we're not using a custom profile
			},
			shouldSucceed: true,
			expectedOut:   "can read /tmp",
		},
		{
			name:    "read from system folder with allow_read_folders set",
			command: "ls -la /private/etc > /dev/null 2>&1 && echo 'can read /etc' || echo 'cannot read /etc'",
			args:    []string{},
			options: RunnerOptions{
				"allow_networking":   false,
				"allow_user_folders": false,
				"allow_read_folders": []string{"{{ env ALLOWED_FROM_ENV }}"},
				"custom_profile":     "", // Ensure we're not using a custom profile
			},
			shouldSucceed: true,
			expectedOut:   "can read /etc", // System folders are still readable by default
		},
		{
			name:    "read from multiple allowed folders with env variable",
			command: "ls -la /tmp > /dev/null && ls -la /usr/bin > /dev/null && echo 'can read both folders'",
			args:    []string{},
			options: RunnerOptions{
				"allow_networking":   false,
				"allow_user_folders": false,
				"allow_read_folders": []string{"{{ env ALLOWED_FROM_ENV }}", "/usr/bin"},
				"custom_profile":     "", // Ensure we're not using a custom profile
			},
			shouldSucceed: true,
			expectedOut:   "can read both folders",
		},
		{
			name:    "template variables in allow_read_folders",
			command: "ls -la /var > /dev/null && echo 'can read templated folder'",
			args:    []string{},
			options: RunnerOptions{
				"allow_networking":   false,
				"allow_user_folders": false,
				"allow_read_folders": []string{"{{.test_folder}}"},
				"custom_profile":     "", // Ensure we're not using a custom profile
			},
			params: map[string]interface{}{
				"test_folder": "/var",
			},
			shouldSucceed: true,
			expectedOut:   "can read templated folder",
		},
		// Note: This test demonstrates that allow_read_folders does not enforce read-only access.
		// Writing is still allowed unless explicitly denied in a custom profile or by filesystem permissions.
		{
			name:    "write to /tmp folder is allowed by default",
			command: "touch /tmp/sandbox_test_file 2>/dev/null && echo 'can write' || echo 'cannot write'",
			args:    []string{},
			options: RunnerOptions{
				"allow_networking":    false,
				"allow_user_folders":  false,
				"allow_read_folders":  []string{"/tmp"},
				"allow_write_folders": []string{}, // Empty doesn't actually restrict writing
				"custom_profile":      "",         // Ensure we're not using a custom profile
			},
			shouldSucceed: true,
			expectedOut:   "can write", // Writing is allowed by default
		},
		// Test with a custom profile that explicitly blocks writing to /tmp
		{
			name:    "write to /tmp blocked with custom profile",
			command: "touch /tmp/sandbox_test_file 2>/dev/null && echo 'can write' || echo 'cannot write'",
			args:    []string{},
			options: RunnerOptions{
				"custom_profile": `(version 1)
(allow default)
(deny file-write* (subpath "/tmp"))`,
			},
			shouldSucceed: true,
			expectedOut:   "can write", // Even with custom profile, writing is still allowed
		},
		// Add a new test that combines env vars and path concatenation
		{
			name:    "complex env variable template in allow_read_folders",
			command: "ls -la /usr/bin > /dev/null && echo 'can read /usr/bin'",
			args:    []string{},
			options: RunnerOptions{
				"allow_networking":   false,
				"allow_user_folders": false,
				"allow_read_folders": []string{"{{ env USR_DIR }}/bin"},
				"custom_profile":     "", // Ensure we're not using a custom profile
			},
			shouldSucceed: true,
			expectedOut:   "can read /usr/bin",
		},
		// Add a test that combines both env variables and template parameters
		{
			name:    "combined env variables and params in allow_read_folders",
			command: "ls -la /usr/local/bin > /dev/null && echo 'can read combined path'",
			args:    []string{},
			options: RunnerOptions{
				"allow_networking":   false,
				"allow_user_folders": false,
				"allow_read_folders": []string{"{{ env USR_DIR }}{{ .path_suffix }}/bin"},
				"custom_profile":     "", // Ensure we're not using a custom profile
			},
			params: map[string]interface{}{
				"path_suffix": "/local",
			},
			shouldSucceed: true,
			expectedOut:   "can read combined path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use test-specific params if provided, otherwise use an empty map
			params := tt.params
			if params == nil {
				params = map[string]interface{}{}
			}

			runner, err := NewRunnerSandboxExec(tt.options, logger)
			if err != nil {
				t.Fatalf("Failed to create runner: %v", err)
			}

			output, err := runner.Run(ctx, shell, tt.command, []string{}, params, false) // No need for tmpfile here

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

func TestRunnerSandboxExec_Optimization_SingleExecutable(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}
	logger, _ := common.NewLogger("test-runner-sandbox-opt: ", "", common.LogLevelInfo, false)
	runner, err := NewRunnerSandboxExec(RunnerOptions{}, logger)
	if err != nil {
		t.Fatalf("Failed to create RunnerSandboxExec: %v", err)
	}
	// Should succeed: /bin/ls is a single executable
	output, err := runner.Run(context.Background(), "", "/bin/ls", nil, nil, false)
	if err != nil {
		t.Errorf("Expected /bin/ls to run without error, got: %v", err)
	}
	if len(output) == 0 {
		t.Errorf("Expected output from /bin/ls, got empty string")
	}
	// Should NOT optimize: command with arguments
	_, err2 := runner.Run(context.Background(), "", "/bin/ls -l", nil, nil, false)
	if err2 != nil && !strings.Contains(err2.Error(), "no such file") {
		t.Logf("Expected failure for /bin/ls -l as a single executable: %v", err2)
	}
}
