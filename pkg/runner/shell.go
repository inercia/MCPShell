package runner

import (
	"os"
	"runtime"
	"strings"
)

// isCmdShell checks if the given shell is a Windows cmd shell
func isCmdShell(shell string) bool {
	shellLower := strings.ToLower(shell)
	return strings.Contains(shellLower, "cmd") || strings.HasSuffix(shellLower, "cmd.exe")
}

// isPowerShell checks if the given shell is a PowerShell
func isPowerShell(shell string) bool {
	shellLower := strings.ToLower(shell)
	return strings.Contains(shellLower, "powershell") || strings.HasSuffix(shellLower, "powershell.exe") ||
		strings.HasSuffix(shellLower, "pwsh.exe")
}

// isWindowsShell checks if the given shell is a Windows-specific shell (cmd or powershell)
func isWindowsShell(shell string) bool {
	return isCmdShell(shell) || isPowerShell(shell)
}

// getShell returns the shell to use for command execution,
// using the provided shell, falling back to $SHELL env var,
// and finally using appropriate default based on OS.
//
// Parameters:
//   - configShell: The configured shell to use (can be empty)
//
// Returns:
//   - The shell executable path to use
func getShell(configShell string) string {
	if configShell != "" {
		return configShell
	}

	// On Windows, default to cmd.exe if SHELL is not set
	if runtime.GOOS == "windows" {
		shell := os.Getenv("COMSPEC") // More reliable on Windows
		if shell != "" {
			return shell
		}
		return "cmd.exe" // Fallback for Windows
	}

	shell := os.Getenv("SHELL")
	if shell != "" {
		return shell
	}

	return "/bin/sh" // Default for Unix-like systems
}
