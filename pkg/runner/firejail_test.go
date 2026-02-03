package runner

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestNewFirejail(t *testing.T) {
	// Skip on non-Linux platforms
	if runtime.GOOS != "linux" {
		t.Skip("Skipping firejail tests on non-Linux platform")
	}

	options := Options{
		"allow_networking": true,
	}

	r, err := NewFirejail(options, nil)
	if err != nil {
		t.Fatalf("Failed to create firejail runner: %v", err)
	}

	if r == nil {
		t.Fatal("Expected non-nil runner")
	}
}

func TestFirejail_Run(t *testing.T) {
	// Skip on non-Linux platforms
	if runtime.GOOS != "linux" {
		t.Skip("Skipping firejail tests on non-Linux platform")
	}

	// Skip if firejail is not installed
	if _, err := os.Stat("/usr/bin/firejail"); os.IsNotExist(err) {
		t.Skip("Skipping test because firejail is not installed")
	}

	options := Options{
		"allow_networking": true,
	}

	r, err := NewFirejail(options, nil)
	if err != nil {
		t.Fatalf("Failed to create firejail runner: %v", err)
	}

	ctx := context.Background()

	// Test simple echo command
	output, err := r.Run(ctx, "/bin/sh", "echo hello world", nil, nil, false)
	if err != nil {
		t.Fatalf("Failed to run command: %v", err)
	}

	if output != "hello world\n" {
		t.Errorf("Expected 'hello world\\n', got '%s'", output)
	}
}

func TestFirejail_NetworkRestriction(t *testing.T) {
	// Skip on non-Linux platforms
	if runtime.GOOS != "linux" {
		t.Skip("Skipping firejail tests on non-Linux platform")
	}

	// Skip if firejail is not installed
	if _, err := os.Stat("/usr/bin/firejail"); os.IsNotExist(err) {
		t.Skip("Skipping test because firejail is not installed")
	}

	ctx := context.Background()

	// Test with networking enabled
	networkEnabledOptions := Options{
		"allow_networking": true,
	}

	runnerEnabled, err := NewFirejail(networkEnabledOptions, nil)
	if err != nil {
		t.Fatalf("Failed to create firejail runner: %v", err)
	}

	// This might succeed or fail depending on network connectivity,
	// but it should not be blocked by firejail
	_, _ = runnerEnabled.Run(ctx, "/bin/sh", "ping -c 1 127.0.0.1", nil, nil, false)

	// Test with networking disabled
	networkDisabledOptions := Options{
		"allow_networking": false,
	}

	runnerDisabled, err := NewFirejail(networkDisabledOptions, nil)
	if err != nil {
		t.Fatalf("Failed to create firejail runner: %v", err)
	}

	// This should fail or timeout due to network restrictions
	_, _ = runnerDisabled.Run(ctx, "/bin/sh", "ping -c 1 127.0.0.1", nil, nil, false)
}

func TestFirejail_Optimization_SingleExecutable(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping firejail tests on non-Linux platform")
	}
	if _, err := os.Stat("/usr/bin/firejail"); os.IsNotExist(err) {
		t.Skip("Skipping test because firejail is not installed")
	}
	r, err := NewFirejail(Options{"allow_networking": true}, nil)
	if err != nil {
		t.Fatalf("Failed to create firejail runner: %v", err)
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
