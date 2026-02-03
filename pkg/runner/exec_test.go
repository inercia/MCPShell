package runner

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/inercia/MCPShell/pkg/common"
)

func TestNewExecOptions(t *testing.T) {
	tests := []struct {
		name    string
		options Options
		want    ExecOptions
		wantErr bool
	}{
		{
			name: "valid options with shell",
			options: Options{
				"shell": "/bin/bash",
			},
			want: ExecOptions{
				Shell: "/bin/bash",
			},
			wantErr: false,
		},
		{
			name:    "empty options",
			options: Options{},
			want:    ExecOptions{},
			wantErr: false,
		},
		{
			name: "options with additional fields",
			options: Options{
				"shell": "/bin/zsh",
				"extra": "value",
			},
			want: ExecOptions{
				Shell: "/bin/zsh",
			},
			wantErr: false,
		},
		{
			name: "options with numeric shell as string",
			options: Options{
				"shell": "123",
			},
			want: ExecOptions{
				Shell: "123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewExecOptions(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewExecOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewExecOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExec_Run(t *testing.T) {
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
			name:    "command with environment variable",
			shell:   "",
			command: "echo $TEST_VAR",
			env:     []string{"TEST_VAR=test_value"},
			params:  nil,
			want:    "test_value",
			wantErr: false,
		},
	}

	if runtime.GOOS == "windows" {
		tests[1].command = "echo %TEST_VAR%"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := common.NewLogger("test-runner-exec: ", "", common.LogLevelInfo, false)
			r, err := NewExec(Options{}, logger)
			if err != nil {
				t.Fatalf("Failed to create Exec: %v", err)
			}

			got, err := r.Run(context.Background(), tt.shell, tt.command, tt.env, tt.params, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exec.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Trim any trailing newlines for comparison
			got = strings.TrimSpace(got)

			if got != tt.want {
				t.Errorf("Exec.Run() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExec_RunWithEnvExpansion(t *testing.T) {
	// This test demonstrates using the -c flag to execute a command with environment variable expansion
	logger, _ := common.NewLogger("test-runner-exec-env: ", "", common.LogLevelInfo, false)

	r, err := NewExec(Options{}, logger)
	if err != nil {
		t.Fatalf("Failed to create Exec: %v", err)
	}

	command := "echo $TEST_VAR"
	if runtime.GOOS == "windows" {
		command = "echo %TEST_VAR%"
	}

	// Use the shell's -c flag directly to execute a command that expands an environment variable
	output, err := r.Run(
		context.Background(),
		"",
		command,
		[]string{"TEST_VAR=test_value_expanded"},
		nil,
		false, // No tmpfile needed for this test
	)

	if err != nil {
		t.Fatalf("Exec.Run() error = %v", err)
	}

	output = strings.TrimSpace(output)
	expected := "test_value_expanded"

	if output != expected {
		t.Errorf("Environment variable expansion failed: got %q, want %q", output, expected)
	}
}
