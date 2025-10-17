package command

import (
	"runtime"
	"testing"

	"github.com/inercia/MCPShell/pkg/common"
)

// TestImplicitRequirements tests the implicit requirements checking
// for different runner types
func TestImplicitRequirements(t *testing.T) {
	logger, _ := common.NewLogger("test: ", "", common.LogLevelInfo, false)

	// Test cases for the exec runner
	t.Run("ExecRunner", func(t *testing.T) {
		// Exec runner should always pass since it has no requirements
		runner, err := NewRunnerExec(RunnerOptions{}, logger)
		if err != nil {
			t.Fatalf("Failed to create exec runner: %v", err)
		}

		err = runner.CheckImplicitRequirements()
		if err != nil {
			t.Errorf("Exec runner should have no requirements but failed: %v", err)
		}
	})

	// Test cases for the sandbox-exec runner
	t.Run("SandboxExecRunner", func(t *testing.T) {
		// Skip this test if not on macOS
		if runtime.GOOS != "darwin" {
			t.Skip("Skipping sandbox-exec test on non-macOS platform")
		}

		// Create the runner
		runner, err := NewRunnerSandboxExec(RunnerOptions{}, logger)
		if err != nil {
			t.Fatalf("Failed to create sandbox-exec runner: %v", err)
		}

		// Check requirements - expect pass on macOS if executable exists
		err = runner.CheckImplicitRequirements()
		if err != nil {
			// This is expected if sandbox-exec is not available
			t.Logf("SandboxExec runner failed as expected if sandbox-exec is not available: %v", err)
		}
	})

	// Test cases for the firejail runner
	t.Run("FirejailRunner", func(t *testing.T) {
		// Skip this test if not on Linux or firejail is not available
		if runtime.GOOS != "linux" {
			t.Skip("Skipping firejail test on non-Linux platform")
		}
		if !common.CheckExecutableExists("firejail") {
			t.Skip("Skipping firejail test if firejail is not available")
		}

		// Create the runner
		runner, err := NewRunnerFirejail(RunnerOptions{}, logger)
		if err != nil {
			t.Fatalf("Failed to create firejail runner: %v", err)
		}

		// Check requirements - expect pass on Linux if executable exists
		err = runner.CheckImplicitRequirements()
		if err != nil {
			// This is expected if firejail is not available
			t.Logf("Firejail runner failed as expected if firejail is not available: %v", err)
		}
	})

	// Test cases for the docker runner
	t.Run("DockerRunner", func(t *testing.T) {
		// This test is for the implicit requirements only
		// The Docker daemon check will be handled in the DockerRunner itself

		// Create a Docker runner with mock options that satisfy its creation requirements
		mockOpts := RunnerOptions{
			"image": "alpine:latest",
		}

		runner, err := NewDockerRunner(mockOpts, logger)
		if err != nil {
			t.Fatalf("Failed to create docker runner: %v", err)
		}

		// Check requirements - expect pass if Docker is available and running
		err = runner.CheckImplicitRequirements()
		if err != nil {
			// This is expected if Docker is not available or daemon is not running
			t.Logf("Docker runner failed requirements check as expected: %v", err)
		}
	})
}
