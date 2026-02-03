package runner

import (
	"os"
	"strings"

	"github.com/inercia/MCPShell/pkg/common"
)

// isSingleExecutableCommand checks if the command string is a single word (no spaces or shell metacharacters)
// and if that word is an existing executable (absolute/relative path or in PATH).
func isSingleExecutableCommand(command string) bool {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return false
	}
	// Disallow spaces, shell metacharacters, and redirections
	if strings.ContainsAny(cmd, " \t|&;<>(){}[]$`'\"\n") {
		return false
	}
	// If it's an absolute or relative path
	if strings.HasPrefix(cmd, "/") || strings.HasPrefix(cmd, ".") {
		info, err := os.Stat(cmd)
		if err != nil {
			return false
		}
		mode := info.Mode()
		return !info.IsDir() && mode&0111 != 0 // executable by someone
	}
	// Otherwise, check if it's in PATH
	return common.CheckExecutableExists(cmd)
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// shellQuote returns a shell-safe quoted string.
// It uses single quotes and escapes any single quotes within the string.
func shellQuote(s string) string {
	// If the string contains no special characters, return as-is
	if !strings.ContainsAny(s, " \t\n'\"\\$`!*?[]{}();<>&|") {
		return s
	}
	// Use single quotes and escape any single quotes by ending the quoted string,
	// adding an escaped single quote, and starting a new quoted string
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
