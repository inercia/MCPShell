package root

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/config"
	"github.com/inercia/MCPShell/pkg/server"
	"github.com/inercia/MCPShell/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	useHTTP  bool
	httpPort int
	daemon   bool
)

// mcpCommand represents the run command which starts the MCP server
var mcpCommand = &cobra.Command{
	Use:     "mcp",
	Aliases: []string{"serve", "server", "run"},
	Short:   "Run the MCP server for a MCP configuration file",
	Long: `
Run an MCP server that provides tools to LLM applications.
This command starts a server that communicates using the Model Context Protocol (MCP)
and expooses the tools defined in a MCP configuration file.

The server loads tool definitions from a MCP configuration file and makes them
available to AI applications via the MCP protocol.

When using --http mode, you can also use --daemon to run the server in the background
and ignore SIGHUP signals.
`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger
		logger, err := initLogger()
		if err != nil {
			return err
		}

		defer common.RecoverPanic()

		logger.Info("Starting MCPShell")

		// Check if config files are provided
		if len(toolsFiles) == 0 {
			logger.Error("Tools configuration file(s) are required")
			return fmt.Errorf("tools configuration file(s) are required. Use --tools flag to specify the path(s)")
		}

		// Ensure tools directory exists
		if err := utils.EnsureToolsDir(); err != nil {
			logger.Error("Failed to ensure tools directory: %v", err)
			return fmt.Errorf("failed to ensure tools directory: %w", err)
		}

		// Daemon mode is only supported with HTTP mode
		if daemon && !useHTTP {
			logger.Error("Daemon mode is only supported with HTTP mode")
			return fmt.Errorf("daemon mode is only supported with HTTP mode (use --http flag)")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := common.GetLogger()
		defer common.RecoverPanic()

		// Daemonize if requested
		if daemon {
			if err := daemonize(); err != nil {
				logger.Error("Failed to daemonize: %v", err)
				return fmt.Errorf("failed to daemonize: %w", err)
			}
			logger.Info("Daemonized successfully")
		}

		// Load the configuration file(s) (local or remote)
		localConfigPath, cleanup, err := config.ResolveMultipleConfigPaths(toolsFiles, logger)
		if err != nil {
			logger.Error("Failed to load configuration: %v", err)
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Ensure temporary files are cleaned up
		defer cleanup()

		// Create and start the server
		srv := server.New(server.Config{
			ConfigFile:          localConfigPath,
			Logger:              logger,
			Version:             version,
			Descriptions:        description,
			DescriptionFiles:    descriptionFile,
			DescriptionOverride: descriptionOverride,
		})

		if useHTTP {
			// Set up SIGHUP handling for daemon mode
			if daemon {
				setupSIGHUPHandler(logger)
			}
			return srv.StartHTTP(httpPort)
		}
		return srv.Start()
	},
}



// setupSIGHUPHandler sets up signal handling to ignore SIGHUP in daemon mode
func setupSIGHUPHandler(logger *common.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)

	go func() {
		for {
			sig := <-sigChan
			if sig == syscall.SIGHUP {
				logger.Info("Received SIGHUP, ignoring in daemon mode")
			}
		}
	}()
}

// init adds flags to the run command
func init() {
	rootCmd.AddCommand(mcpCommand)

	// Add MCP-specific flags
	mcpCommand.Flags().StringSliceVarP(&description, "description", "d", []string{}, "MCP server description (optional, can be specified multiple times)")
	mcpCommand.Flags().StringSliceVarP(&descriptionFile, "description-file", "", []string{}, "Read the MCP server description from files (optional, can be specified multiple times)")
	mcpCommand.Flags().BoolVarP(&descriptionOverride, "description-override", "", false, "Override the description found in the config file")

	// Add HTTP server flags
	mcpCommand.Flags().BoolVar(&useHTTP, "http", false, "Enable HTTP server mode (serve MCP over HTTP/SSE instead of stdio)")
	mcpCommand.Flags().IntVar(&httpPort, "port", 8080, "Port for HTTP server (default: 8080, only used with --http)")
	mcpCommand.Flags().BoolVar(&daemon, "daemon", false, "Run in daemon mode (background process, ignores SIGHUP, only works with --http)")

	// Mark required flags
	_ = mcpCommand.MarkFlagRequired("tools")
}
