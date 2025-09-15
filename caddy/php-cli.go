package caddy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	caddycmd "github.com/caddyserver/caddy/v2/cmd"
	"github.com/dunglas/frankenphp"

	"github.com/spf13/cobra"
)

func init() {
	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "php-cli",
		Usage: "[-d key=value] script.php [args ...]",
		Short: "Runs a PHP command",
		Long: `
Executes a PHP script similarly to the CLI SAPI.

The -d option allows you to define INI entries at runtime, similar to the standard PHP CLI.
Example: frankenphp php-cli -d memory_limit=256M script.php`,
		CobraFunc: func(cmd *cobra.Command) {
			cmd.Flags().StringArrayP("define", "d", []string{}, "Define INI entry (key=value)")
			// Disable Cobra's flag parsing to handle PHP CLI compatibility manually
			cmd.DisableFlagParsing = true
			cmd.RunE = caddycmd.WrapCommandFuncForCobra(cmdPHPCLI)
		},
	})
}

func cmdPHPCLI(fs caddycmd.Flags) (int, error) {
	// Get all arguments after "php-cli"
	allArgs := os.Args[2:] // Skip "frankenphp" and "php-cli"

	// Manually parse -d flags and collect remaining args
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

	if len(args) < 1 {
		return 1, errors.New("the path to the PHP script is required")
	}

	// Handle embedded app path
	if frankenphp.EmbeddedAppPath != "" {
		if _, err := os.Stat(args[0]); err != nil {
			args[0] = filepath.Join(frankenphp.EmbeddedAppPath, args[0])
		}
	}

	var status int
	if len(args) >= 2 && args[0] == "-r" {
		// For -r flag, wrap the code with ini_set calls if needed
		phpCode := args[1]
		if len(phpIni) > 0 {
			var iniCalls []string
			for key, value := range phpIni {
				iniCalls = append(iniCalls, fmt.Sprintf("ini_set('%s', '%s');",
					strings.ReplaceAll(key, "'", "\\'"),
					strings.ReplaceAll(value, "'", "\\'")))
			}
			phpCode = strings.Join(iniCalls, " ") + " " + phpCode
		}
		status = frankenphp.ExecutePHPCode(phpCode)
	} else {
		// For script files, we need a different approach
		if len(phpIni) > 0 {
			// Create a temporary wrapper script that sets INI values and includes the original
			var iniCalls []string
			for key, value := range phpIni {
				iniCalls = append(iniCalls, fmt.Sprintf("ini_set('%s', '%s');",
					strings.ReplaceAll(key, "'", "\\'"),
					strings.ReplaceAll(value, "'", "\\'")))
			}

			wrapperCode := "<?php " + strings.Join(iniCalls, " ") +
				fmt.Sprintf(" require_once '%s';", strings.ReplaceAll(args[0], "'", "\\'"))

			// Write to temporary file
			tmpFile, err := os.CreateTemp("", "frankenphp-cli-*.php")
			if err != nil {
				return 1, fmt.Errorf("failed to create temporary file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			defer tmpFile.Close()

			if _, err := tmpFile.WriteString(wrapperCode); err != nil {
				return 1, fmt.Errorf("failed to write wrapper script: %v", err)
			}
			tmpFile.Close()

			// Execute the wrapper script
			wrapperArgs := make([]string, len(args))
			wrapperArgs[0] = tmpFile.Name()
			copy(wrapperArgs[1:], args[1:])
			status = frankenphp.ExecuteScriptCLI(tmpFile.Name(), wrapperArgs)
		} else {
			status = frankenphp.ExecuteScriptCLI(args[0], args)
		}
	}

	os.Exit(status)

	return status, nil
}
