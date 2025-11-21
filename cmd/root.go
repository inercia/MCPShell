// Package root contains the command-line interface implementation for the MCPShell.
//
// It defines the root command and all subcommands using Cobra and manages CLI flags,
// execution flow, and global application state.
package root

import (
	"fmt"
	"os"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/spf13/cobra"
)

// ApplicationName is the name of the application used in various places
const ApplicationName = "mcpshell"

// Common command-line flags
var (
	// Common flags
	toolsFiles []string
	logFile    string
	logLevel   string
	verbose    bool

	// MCP server flags
	description         []string
	descriptionFile     []string
	descriptionOverride bool

	// Agent-specific flags
	agentModel        string
	agentSystemPrompt string
	agentUserPrompt   string
	agentOpenAIApiKey string
	agentOpenAIApiURL string
	agentOnce         bool

	// Application version information (set via SetVersion from main)
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   ApplicationName,
	Short: "MCPShell",
	Long: `
MCPShell is a MCP bridge for LLMs and shell commands.

This CLI application enables AI systems to securely execute commands through
the Model Context Protocol (MCP).

Specify your tools configuration using the --tools flag (supports multiple files):
  mcpshell --tools /path/to/tools.yaml              (single file, absolute path)
  mcpshell --tools mytools                          (single file in global tools directory, adds .yaml)
  mcpshell --tools mytools.yaml                     (single file in tools directory)
  mcpshell --tools /some/dir                        (load all tools in the directory)
  mcpshell --tools file1.yaml --tools file2.yaml    (multiple files)
  mcpshell --tools file1.yaml,file2.yaml            (multiple files, comma-separated)
  
The tools directory defaults to ~/.mcpshell/tools but can be overridden 
with the MCPSHELL_TOOLS_DIR environment variable.

When multiple configuration files are provided, they are merged with:

- Prompts concatenated from all files
- Tools combined from all files  
- MCP description and run config taken from the first file
`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is specified, show the help
		_ = cmd.Help()
	},
}

// SetVersion sets the version information from build-time variables
func SetVersion(v, c, d string) {
	if v != "" {
		version = v
	}
	if c != "" {
		commit = c
	}
	if d != "" {
		buildDate = d
	}
}

// GetVersion returns the full version string with build information
func GetVersion() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, buildDate)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	defer common.RecoverPanic()

	if err := rootCmd.Execute(); err != nil {
		common.GetLogger().Error("Command execution failed: %v", err)
		fmt.Println(err)
		os.Exit(1)
	}
}

// init registers all subcommands and sets up global flags
func init() {
	// Add common persistent flags
	rootCmd.PersistentFlags().StringSliceVar(&toolsFiles, "tools", []string{}, "Path(s) to the tools configuration file(s).\nSupports multiple files via --tools=file1 --tools=file2 or --tools=file1,file2.\nEach path supports relative paths and auto .yaml extension.\nDefault look path from MCPSHELL_TOOLS_DIR")
	rootCmd.PersistentFlags().StringVarP(&logFile, "logfile", "l", "", "Path to the log file (optional)")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "", "info", "Log level: none, error, info, debug")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging (sets log level to debug)")

	// Add version flag to all commands with custom handling
	versionFlag := false
	rootCmd.PersistentFlags().BoolVar(&versionFlag, "version", false, "Print version information")

	// Add pre-run to handle version flag
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
			fmt.Println(GetVersion())
			os.Exit(0)
		}
	}
}

// initLogger initializes the logger with the specified configuration
func initLogger() (*common.Logger, error) {
	// If verbose flag is set, use debug level; otherwise use the configured log level
	var level common.LogLevel
	if verbose {
		level = common.LogLevelDebug
	} else {
		level = common.LogLevelFromString(logLevel)
	}

	logger, err := common.NewLogger("[mcpshell] ", logFile, level, true)
	if err != nil {
		return nil, fmt.Errorf("failed to set up logger: %w", err)
	}

	common.SetLogger(logger)
	return logger, nil
}
