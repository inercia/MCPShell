package config

import (
	"runtime"
	"testing"
)

func TestCheckToolPrerequisites(t *testing.T) {
	// Create test cases
	tests := []struct {
		name     string
		runners  []MCPToolRunner
		expected bool
	}{
		{
			name:     "No runners (default exec runner)",
			runners:  nil,
			expected: true,
		},
		{
			name: "One runner with matching OS",
			runners: []MCPToolRunner{
				{
					Name: "compatible-runner",
					Requirements: MCPToolRequirements{
						OS: runtime.GOOS,
					},
				},
			},
			expected: true,
		},
		{
			name: "One runner with non-matching OS",
			runners: []MCPToolRunner{
				{
					Name: "incompatible-runner",
					Requirements: MCPToolRequirements{
						OS: "non-existent-os",
					},
				},
			},
			expected: false,
		},
		{
			name: "One runner with existing executable",
			runners: []MCPToolRunner{
				{
					Name: "compatible-runner",
					Requirements: MCPToolRequirements{
						Executables: []string{"sh"}, // should exist on most systems
					},
				},
			},
			expected: true,
		},
		{
			name: "One runner with non-existent executable",
			runners: []MCPToolRunner{
				{
					Name: "incompatible-runner",
					Requirements: MCPToolRequirements{
						Executables: []string{"non-existent-executable-12345"},
					},
				},
			},
			expected: false,
		},
		{
			name: "Multiple runners with one compatible",
			runners: []MCPToolRunner{
				{
					Name: "incompatible-runner",
					Requirements: MCPToolRequirements{
						OS: "non-existent-os",
					},
				},
				{
					Name: "compatible-runner",
					Requirements: MCPToolRequirements{
						OS: runtime.GOOS,
					},
				},
			},
			expected: true,
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := Tool{
				Config: MCPToolConfig{
					Run: MCPToolRunConfig{
						Runners: tt.runners,
					},
				},
			}
			result := tool.CheckToolRequirements()
			if result != tt.expected {
				t.Errorf("CheckToolRequirements() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCreateTools_Prerequisites(t *testing.T) {
	// Create a simple config with two tools, one with unmet prerequisites
	cfg := &ToolsConfig{
		MCP: MCPConfig{
			Tools: []MCPToolConfig{
				{
					Name:        "tool1",
					Description: "Tool with met prerequisites",
					Run: MCPToolRunConfig{
						Command: "echo 'Tool 1'",
						Runners: []MCPToolRunner{
							{
								Name: "compatible-runner",
								Requirements: MCPToolRequirements{
									OS:          runtime.GOOS,
									Executables: []string{"sh"}, // should exist on most systems
								},
							},
						},
					},
				},
				{
					Name:        "tool2",
					Description: "Tool with unmet prerequisites",
					Run: MCPToolRunConfig{
						Command: "echo 'Tool 2'",
						Runners: []MCPToolRunner{
							{
								Name: "incompatible-runner",
								Requirements: MCPToolRequirements{
									OS:          "non-existent-os",
									Executables: []string{"non-existent-executable-12345"},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create tools
	tools := cfg.GetTools()

	// We should have only one tool
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}

	// Check that the correct tool was created
	if len(tools) > 0 && tools[0].MCPTool.Name != "tool1" {
		t.Errorf("Expected tool named 'tool1', got '%s'", tools[0].MCPTool.Name)
	}
}
