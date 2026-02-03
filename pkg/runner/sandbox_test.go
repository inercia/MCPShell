package runner

import (
	"context"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/inercia/MCPShell/pkg/common"
)

func TestNewSandboxExecOptions(t *testing.T) {
	// Skip on non-macOS platforms
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	tests := []struct {
		name    string
		options Options
		want    SandboxExecOptions
		wantErr bool
	}{
		{
			name: "valid options with all fields",
			options: Options{
				"shell":              "/bin/bash",
				"allow_networking":   true,
				"allow_user_folders": true,
				"custom_profile":     "(version 1)(allow default)",
			},
			want: SandboxExecOptions{
				Shell:            "/bin/bash",
				AllowNetworking:  true,
				AllowUserFolders: true,
				CustomProfile:    "(version 1)(allow default)",
			},
			wantErr: false,
		},
		{
			name:    "empty options",
			options: Options{},
			want:    SandboxExecOptions{},
			wantErr: false,
		},
		{
			name: "options with partial fields",
			options: Options{
				"shell":            "/bin/zsh",
				"allow_networking": false,
			},
			want: SandboxExecOptions{
				Shell:           "/bin/zsh",
				AllowNetworking: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSandboxExecOptions(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSandboxExecOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSandboxExecOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

// This test is only run on macOS as it requires sandbox-exec
func TestSandboxExec_Run(t *testing.T) {
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
		_ = os.Unsetenv("ALLOWED_FROM_ENV")
		_ = os.Unsetenv("USR_DIR")
	}()

	// Create a logger for the test
	logger, _ := common.NewLogger("test-runner-sandbox: ", "", common.LogLevelInfo, false)
	ctx := context.Background()
	shell := "" // use default

	tests := []struct {
		name          string
		command       string
		options       Options
		params        map[string]interface{}
		shouldSucceed bool
		expectedOut   string
	}{
		{
			name:    "echo command with full permissions",
			command: "echo 'Hello Sandbox'",
			options: Options{
				"allow_networking":   true,
				"allow_user_folders": true,
			},
			shouldSucceed: true,
			expectedOut:   "Hello Sandbox",
		},
		{
			name:    "echo command with networking disabled",
			command: "echo 'No Network'",
			options: Options{
				"allow_networking":   false,
				"allow_user_folders": true,
			},
			shouldSucceed: true,
			expectedOut:   "No Network",
		},
		{
			name:    "echo command with all restrictions",
			command: "echo 'Restricted'",
			options: Options{
				"allow_networking":   false,
				"allow_user_folders": false,
			},
			shouldSucceed: true,
			expectedOut:   "Restricted",
		},
		{
			name:    "read /tmp with folder restrictions",
			command: "ls -la /tmp | grep -q . && echo 'success'",
			options: Options{
				"allow_networking":   false,
				"allow_user_folders": false,
			},
			shouldSucceed: true,
			expectedOut:   "success",
		},
		{
			name:    "custom profile allowing only /tmp",
			command: "ls -la /tmp | grep -q . && echo 'success'",
			options: Options{
				"custom_profile": `(version 1)
(allow default)
(deny file-read* (subpath "/Users"))
(allow file-read* (regex "^/tmp"))`,
			},
			shouldSucceed: true,
			expectedOut:   "success",
		},
		{
			name:    "read from allowed folder using env variable",
			command: "ls -la /tmp > /dev/null && echo 'can read /tmp'",
			options: Options{
				"allow_networking":   false,
				"allow_user_folders": false,
				"allow_read_folders": []string{"{{ env ALLOWED_FROM_ENV }}"},
				"custom_profile":     "",
			},
			shouldSucceed: true,
			expectedOut:   "can read /tmp",
		},
		{
			name:    "template variables in allow_read_folders",
			command: "ls -la /var > /dev/null && echo 'can read templated folder'",
			options: Options{
				"allow_networking":   false,
				"allow_user_folders": false,
				"allow_read_folders": []string{"{{.test_folder}}"},
				"custom_profile":     "",
			},
			params: map[string]interface{}{
				"test_folder": "/var",
			},
			shouldSucceed: true,
			expectedOut:   "can read templated folder",
		},
		{
			name:    "complex env variable template in allow_read_folders",
			command: "ls -la /usr/bin > /dev/null && echo 'can read /usr/bin'",
			options: Options{
				"allow_networking":   false,
				"allow_user_folders": false,
				"allow_read_folders": []string{"{{ env USR_DIR }}/bin"},
				"custom_profile":     "",
			},
			shouldSucceed: true,
			expectedOut:   "can read /usr/bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.params
			if params == nil {
				params = map[string]interface{}{}
			}

			r, err := NewSandboxExec(tt.options, logger)
			if err != nil {
				t.Fatalf("Failed to create runner: %v", err)
			}

			output, err := r.Run(ctx, shell, tt.command, []string{}, params, false)

			if tt.shouldSucceed && err != nil {
				t.Errorf("Expected command to succeed but got error: %v", err)
				return
			}

			if !tt.shouldSucceed && err == nil {
				t.Errorf("Expected command to fail but it succeeded with output: %s", output)
				return
			}

			if tt.shouldSucceed && tt.expectedOut != "" && output != tt.expectedOut {
				t.Errorf("Output mismatch: got %v, want %v", output, tt.expectedOut)
			}
		})
	}
}

func TestSandboxExec_Optimization_SingleExecutable(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}
	logger, _ := common.NewLogger("test-runner-sandbox-opt: ", "", common.LogLevelInfo, false)
	r, err := NewSandboxExec(Options{}, logger)
	if err != nil {
		t.Fatalf("Failed to create SandboxExec: %v", err)
	}
	// Should succeed: /bin/ls is a single executable
	output, err := r.Run(context.Background(), "", "/bin/ls", nil, nil, false)
	if err != nil {
		t.Errorf("Expected /bin/ls to run without error, got: %v", err)
	}
	if len(output) == 0 {
		t.Errorf("Expected output from /bin/ls, got empty string")
	}
	// Should NOT optimize: command with arguments
	_, err2 := r.Run(context.Background(), "", "/bin/ls -l", nil, nil, false)
	if err2 != nil && !strings.Contains(err2.Error(), "no such file") {
		t.Logf("Expected failure for /bin/ls -l as a single executable: %v", err2)
	}
}
