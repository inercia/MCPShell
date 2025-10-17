package command

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/inercia/MCPShell/pkg/common"
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

func TestRunnerExec_Run(t *testing.T) {
	tests := []struct {
		name    string
		shell   string
		command string
		env     []string
		params  map[string]interface{}
		want    string
		wantErr bool
	}{
		{
			name:    "simple echo command",
			shell:   "",
			command: "echo hello world",
			env:     nil,
			params:  nil,
			want:    "hello world",
			wantErr: false,
		},
		{
			name:    "command with special characters",
			shell:   "",
			command: "echo 'hello world with special chars: >& !'",
			env:     nil,
			params:  nil,
			want:    "hello world with special chars: >& !",
			wantErr: false,
		},
		{
			name:    "command with environment variable",
			shell:   "",
			command: "echo $TEST_VAR",
			env:     []string{"TEST_VAR=test_value"},
			params:  nil,
			want:    "test_value",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := common.NewLogger("test-runner-exec: ", "", common.LogLevelInfo, false)
			r, err := NewRunnerExec(RunnerOptions{}, logger)
			if err != nil {
				t.Fatalf("Failed to create RunnerExec: %v", err)
			}

			got, err := r.Run(context.Background(), tt.shell, tt.command, tt.env, tt.params, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunnerExec.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Trim any trailing newlines for comparison
			got = strings.TrimSpace(got)

			if got != tt.want {
				t.Errorf("RunnerExec.Run() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunnerExec_RunWithEnvExpansion(t *testing.T) {
	// This test demonstrates using the -c flag to execute a command with environment variable expansion
	logger, _ := common.NewLogger("test-runner-exec-env: ", "", common.LogLevelInfo, false)

	r, err := NewRunnerExec(RunnerOptions{}, logger)
	if err != nil {
		t.Fatalf("Failed to create RunnerExec: %v", err)
	}

	// Use the shell's -c flag directly to execute a command that expands an environment variable
	output, err := r.Run(
		context.Background(),
		"",
		"echo $TEST_VAR",
		[]string{"TEST_VAR=test_value_expanded"},
		nil,
		false, // No tmpfile needed for this test
	)

	if err != nil {
		t.Fatalf("RunnerExec.Run() error = %v", err)
	}

	output = strings.TrimSpace(output)
	expected := "test_value_expanded"

	if output != expected {
		t.Errorf("Environment variable expansion failed: got %q, want %q", output, expected)
	}
}

func TestRunnerExec_Optimization_SingleExecutable(t *testing.T) {
	logger, _ := common.NewLogger("test-runner-exec-opt: ", "", common.LogLevelInfo, false)
	r, err := NewRunnerExec(RunnerOptions{}, logger)
	if err != nil {
		t.Fatalf("Failed to create RunnerExec: %v", err)
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
		// It's ok if it fails due to the command not existing, but it should not optimize
		t.Logf("Expected failure for /bin/ls -l as a single executable: %v", err2)
	}
}
