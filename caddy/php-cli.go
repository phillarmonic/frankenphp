package caddy

import (
	"errors"
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
			cmd.RunE = caddycmd.WrapCommandFuncForCobra(cmdPHPCLI)
		},
	})
}

func cmdPHPCLI(fs caddycmd.Flags) (int, error) {
	// Parse -d flags
	defines, err := fs.GetStringArray("define")
	if err != nil {
		return 1, err
	}

	// Convert -d flags to ini settings
	phpIni := make(map[string]string)
	for _, define := range defines {
		if key, value, found := strings.Cut(define, "="); found {
			phpIni[key] = value
		} else {
			// Boolean flags default to "1" (enabled)
			phpIni[define] = "1"
		}
	}

	// Get remaining arguments (script and its args)
	// Skip program name and command name to get actual arguments
	allArgs := os.Args[2:] // Skip "frankenphp" and "php-cli"

	// Filter out -d flags and their values to get script and script args
	var args []string
	for i := 0; i < len(allArgs); i++ {
		if allArgs[i] == "-d" || allArgs[i] == "--define" {
			// Skip the flag and its value
			i++ // Skip the value
		} else if strings.HasPrefix(allArgs[i], "-d=") || strings.HasPrefix(allArgs[i], "--define=") {
			// Skip combined flag=value
			continue
		} else {
			// This is a script argument
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

	// Initialize FrankenPHP with custom ini settings if any
	if len(phpIni) > 0 {
		if err := frankenphp.Init(frankenphp.WithPhpIni(phpIni)); err != nil {
			return 1, err
		}
		defer frankenphp.Shutdown()
	}

	var status int
	if len(args) >= 2 && args[0] == "-r" {
		status = frankenphp.ExecutePHPCode(args[1])
	} else {
		status = frankenphp.ExecuteScriptCLI(args[0], args)
	}

	os.Exit(status)

	return status, nil
}
