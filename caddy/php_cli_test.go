package caddy

import (
	"strings"
	"testing"

	caddycmd "github.com/caddyserver/caddy/v2/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestPHPCLIFlagParsing(t *testing.T) {
	// Test parsing various -d flag combinations
	tests := []struct {
		name     string
		args     []string
		expected map[string]string
	}{
		{
			name: "single key=value",
			args: []string{"-d", "memory_limit=256M", "script.php"},
			expected: map[string]string{
				"memory_limit": "256M",
			},
		},
		{
			name: "multiple key=value pairs",
			args: []string{"-d", "memory_limit=256M", "-d", "display_errors=1", "script.php"},
			expected: map[string]string{
				"memory_limit":   "256M",
				"display_errors": "1",
			},
		},
		{
			name: "boolean flag without value",
			args: []string{"-d", "display_errors", "script.php"},
			expected: map[string]string{
				"display_errors": "1",
			},
		},
		{
			name: "mixed flags",
			args: []string{"-d", "memory_limit=128M", "-d", "display_errors", "-d", "max_execution_time=30", "script.php"},
			expected: map[string]string{
				"memory_limit":       "128M",
				"display_errors":     "1",
				"max_execution_time": "30",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh command for each test to avoid flag accumulation
			testCmd := &cobra.Command{
				Use: "test",
			}
			testCmd.Flags().StringArrayP("define", "d", []string{}, "Define INI entry (key=value)")

			// Parse the arguments
			testCmd.SetArgs(tt.args)
			err := testCmd.Execute()
			assert.NoError(t, err)

			// Get the parsed values
			defines, err := testCmd.Flags().GetStringArray("define")
			assert.NoError(t, err)

			// Convert to map like our implementation does
			phpIni := make(map[string]string)
			for _, define := range defines {
				if key, value, found := strings.Cut(define, "="); found {
					phpIni[key] = value
				} else {
					phpIni[define] = "1"
				}
			}

			// Verify the results
			assert.Equal(t, tt.expected, phpIni)
		})
	}
}

func TestPHPCLIUsageString(t *testing.T) {
	// Verify that our usage string includes the -d flag
	commands := caddycmd.Commands()
	foundCommand, exists := commands["php-cli"]

	assert.True(t, exists, "php-cli command should be registered")
	assert.Contains(t, foundCommand.Usage, "[-d key=value]", "Usage should mention -d flag")
	assert.Contains(t, foundCommand.Long, "-d option", "Long description should mention -d option")
}
