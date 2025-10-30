//go:build windows
// +build windows

package command

import (
	"runtime"
	"strings"
)

// getShellCommandArgs returns the correct arguments for different shell types on Windows
func getShellCommandArgs(shell string, command string) (string, []string) {
	shellLower := strings.ToLower(shell)
	
	// Check if this is a cmd shell (Windows)
	if strings.Contains(shellLower, "cmd") || 
	   strings.HasSuffix(shellLower, "cmd.exe") ||
	   (shell == "" && runtime.GOOS == "windows") { // Default to cmd on Windows if no shell specified
		return shell, []string{"/c", command}
	}
	
	// Check if this is a PowerShell
	if strings.Contains(shellLower, "powershell") || 
	   strings.HasSuffix(shellLower, "powershell.exe") ||
	   strings.HasSuffix(shellLower, "pwsh.exe") {
		return shell, []string{"-Command", command}
	}
	
	// For WSL, we might have bash or other Unix shells
	// For Unix-like systems and default fallback
	return shell, []string{"-c", command}
}

// shouldUseUnixTimeoutCommand returns whether to use the Unix-style timeout command
func shouldUseUnixTimeoutCommand() bool {
	// On Windows, we don't use Unix-style timeout command even if a 'timeout' command exists
	// because Windows 'timeout' is for pausing, not for limiting execution time
    return false
}
