// Package utils provides utility functions for MCPShell
package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// MCPShellDirEnv is the environment variable that specifies the configuration directory for MCPShell
	MCPShellDirEnv = "MCPSHELL_DIR"
	// MCPShellToolsDirEnv is the environment variable that specifies the tools directory for MCPShell
	MCPShellToolsDirEnv = "MCPSHELL_TOOLS_DIR"
	// MCPShellHome is the name of the configuration directory for MCPShell
	MCPShellHome = ".mcpshell"
	// MCPShellToolsDir is the name of the tools directory within MCPShell home
	MCPShellToolsDir = "tools"
)

// GetHome returns the user's home directory in a portable way
func GetHome() (string, error) {
	var home string

	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
		if home == "" {
			home = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		}
	} else {
		home = os.Getenv("HOME")
	}

	if home == "" {
		return "", fmt.Errorf("unable to determine home directory")
	}

	return home, nil
}

// GetMCPShellHome returns the MCPShell configuration directory
// This is typically ~/.mcpshell on Unix-like systems or %USERPROFILE%\.mcpshell on Windows
func GetMCPShellHome() (string, error) {
	if mcpHome := os.Getenv(MCPShellDirEnv); mcpHome != "" {
		return mcpHome, nil
	}

	home, err := GetHome()
	if err != nil {
		return "", err
	}

	mcpShellHome := filepath.Join(home, MCPShellHome)
	return mcpShellHome, nil
}

// GetMCPShellToolsDir returns the MCPShell tools directory
// This is typically ~/.mcpshell/tools on Unix-like systems or %USERPROFILE%\.mcpshell\tools on Windows
func GetMCPShellToolsDir() (string, error) {
	if toolsDir := os.Getenv(MCPShellToolsDirEnv); toolsDir != "" {
		return toolsDir, nil
	}

	mcpShellHome, err := GetMCPShellHome()
	if err != nil {
		return "", err
	}

	toolsDir := filepath.Join(mcpShellHome, MCPShellToolsDir)
	return toolsDir, nil
}
