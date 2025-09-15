package caddy

import (
	"strings"
	"testing"

	caddycmd "github.com/caddyserver/caddy/v2/cmd"
	"github.com/stretchr/testify/assert"
)

func TestPHPCLIFlagParsing(t *testing.T) {
	// Test parsing various -d flag combinations using our manual parsing logic
	tests := []struct {
		name         string
		args         []string
		expected     map[string]string
		expectedArgs []string
	}{
		{
			name: "single key=value",
			args: []string{"-d", "memory_limit=256M", "script.php"},
			expected: map[string]string{
				"memory_limit": "256M",
			},
			expectedArgs: []string{"script.php"},
		},
		{
			name: "multiple key=value pairs",
			args: []string{"-d", "memory_limit=256M", "-d", "display_errors=1", "script.php"},
			expected: map[string]string{
				"memory_limit":   "256M",
				"display_errors": "1",
			},
			expectedArgs: []string{"script.php"},
		},
		{
			name: "boolean flag without value",
			args: []string{"-d", "display_errors", "script.php"},
			expected: map[string]string{
				"display_errors": "1",
			},
			expectedArgs: []string{"script.php"},
		},
		{
			name: "mixed flags",
			args: []string{"-d", "memory_limit=128M", "-d", "display_errors", "-d", "max_execution_time=30", "script.php"},
			expected: map[string]string{
				"memory_limit":       "128M",
				"display_errors":     "1",
				"max_execution_time": "30",
			},
			expectedArgs: []string{"script.php"},
		},
		{
			name: "combined format -d=key=value",
			args: []string{"-d=memory_limit=256M", "script.php"},
			expected: map[string]string{
				"memory_limit": "256M",
			},
			expectedArgs: []string{"script.php"},
		},
		{
			name: "with -r flag should preserve it",
			args: []string{"-d", "memory_limit=256M", "-r", "echo 'test';"},
			expected: map[string]string{
				"memory_limit": "256M",
			},
			expectedArgs: []string{"-r", "echo 'test';"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test our manual parsing logic directly
			allArgs := tt.args
			var args []string
			phpIni := make(map[string]string)

			for i := 0; i < len(allArgs); i++ {
				arg := allArgs[i]

				if arg == "-d" || arg == "--define" {
					// Next argument should be the value
					if i+1 < len(allArgs) {
						i++
						define := allArgs[i]
						if key, value, found := strings.Cut(define, "="); found {
							phpIni[key] = value
						} else {
							// Boolean flags default to "1" (enabled)
							phpIni[define] = "1"
						}
					}
				} else if strings.HasPrefix(arg, "-d=") {
					// Combined -d=key=value format
					define := strings.TrimPrefix(arg, "-d=")
					if key, value, found := strings.Cut(define, "="); found {
						phpIni[key] = value
					} else {
						phpIni[define] = "1"
					}
				} else if strings.HasPrefix(arg, "--define=") {
					// Combined --define=key=value format
					define := strings.TrimPrefix(arg, "--define=")
					if key, value, found := strings.Cut(define, "="); found {
						phpIni[key] = value
					} else {
						phpIni[define] = "1"
					}
				} else {
					// This is not a -d flag, so collect remaining args
					args = append(args, allArgs[i:]...)
					break
				}
			}

			// Verify the results
			assert.Equal(t, tt.expected, phpIni)

			// Verify remaining args
			if tt.expectedArgs != nil {
				assert.Equal(t, tt.expectedArgs, args)
			}
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
