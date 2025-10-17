package server

import (
	"fmt"
	"os"
	"strings"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/config"
)

// GetDescription returns the description for the MCP server
// It can get the description from:
// 1. The config file
// 2. Command line flags
// 3. Files
// 4. URLs
func GetDescription(cfg Config) (string, error) {
	var finalDesc string

	// First, check if we should load the description from the config file
	// (unless description override is explicitly requested)
	configDesc := ""
	if !cfg.DescriptionOverride {
		loadedCfg, loadErr := config.NewConfigFromFile(cfg.ConfigFile)
		if loadErr == nil && loadedCfg.MCP.Description != "" {
			configDesc = loadedCfg.MCP.Description
			if cfg.Logger != nil {
				cfg.Logger.Debug("Found description in config file: %s", configDesc)
			}
			finalDesc = configDesc
			if cfg.Logger != nil {
				cfg.Logger.Debug("Using description from config file: %s", configDesc)
			}
		}
	}

	// Add descriptions from command line flags
	if len(cfg.Descriptions) > 0 {
		cmdDesc := strings.Join(cfg.Descriptions, "\n")
		if finalDesc != "" && !cfg.DescriptionOverride {
			finalDesc += "\n" + cmdDesc
			if cfg.Logger != nil {
				cfg.Logger.Info("Appending descriptions from command line flags")
			}
		} else {
			finalDesc = cmdDesc
			if cfg.Logger != nil {
				cfg.Logger.Info("Using descriptions from command line flags")
			}
		}
	}

	// Add descriptions from files - handle both local files and URLs
	if len(cfg.DescriptionFiles) > 0 {
		if cfg.Logger != nil {
			cfg.Logger.Info("Reading server description from files/URLs: %v", cfg.DescriptionFiles)
		}
		var fileDescs []string

		for _, fileOrURL := range cfg.DescriptionFiles {
			var content []byte
			var err error

			// Check if it's a URL
			if strings.HasPrefix(fileOrURL, "http://") || strings.HasPrefix(fileOrURL, "https://") {
				// It's a URL, download the content
				if cfg.Logger != nil {
					cfg.Logger.Info("Downloading description from URL: %s", fileOrURL)
				}
				content, err = common.FetchURLText(fileOrURL)
				if err != nil {
					if cfg.Logger != nil {
						cfg.Logger.Error("Failed to download content from URL: %s - %v", fileOrURL, err)
					}
					return "", fmt.Errorf("failed to download content from URL %s: %w", fileOrURL, err)
				}
			} else {
				// It's a local file, read it directly
				if cfg.Logger != nil {
					cfg.Logger.Info("Reading description from file: %s", fileOrURL)
				}
				content, err = os.ReadFile(fileOrURL)
				if err != nil {
					if cfg.Logger != nil {
						cfg.Logger.Error("Failed to read description file: %s - %v", fileOrURL, err)
					}
					return "", fmt.Errorf("failed to read description file %s: %w", fileOrURL, err)
				}
			}

			fileDescs = append(fileDescs, string(content))
		}

		// Concatenate all file contents
		if len(fileDescs) > 0 {
			fileContent := strings.Join(fileDescs, "\n")
			if finalDesc != "" && !cfg.DescriptionOverride {
				finalDesc += "\n" + fileContent
				if cfg.Logger != nil {
					cfg.Logger.Info("Appending descriptions from files")
				}
			} else {
				finalDesc = fileContent
				if cfg.Logger != nil {
					cfg.Logger.Info("Using descriptions from files")
				}
			}
		}
	}

	return finalDesc, nil
}
