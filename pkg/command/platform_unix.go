//go:build !windows

package command

import (
	"strings"

	"github.com/inercia/MCPShell/pkg/common"
)

// getShellCommandArgs returns the correct arguments for different shell types on Unix systems
func getShellCommandArgs(shell string, command string) (string, []string) {
	shellLower := strings.ToLower(shell)

	// Check if this is a PowerShell (might be available on Unix via PowerShell Core)
	if strings.Contains(shellLower, "powershell") ||
		strings.HasSuffix(shellLower, "powershell.exe") ||
		strings.HasSuffix(shellLower, "pwsh.exe") {
		return shell, []string{"-Command", command}
	}

	// For Unix-like systems and default fallback
	return shell, []string{"-c", command}
}

// shouldUseUnixTimeoutCommand returns whether to use the Unix-style timeout command
func shouldUseUnixTimeoutCommand() bool {
	return common.CheckExecutableExists("timeout")
}
