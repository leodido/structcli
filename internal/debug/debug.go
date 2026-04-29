package internaldebug

import (
	"fmt"
	"os"
	"strings"

	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
)

const (
	FlagAnnotation = "leodido/structcli/debug-flag"
)

// normalizeFormat converts a raw debug flag/env value to "text", "json", or "".
// Truthy values like "true", "1", "yes" are treated as "text" for backward
// compatibility with the old bool flag.
func normalizeFormat(raw string) string {
	v := strings.TrimSpace(strings.ToLower(raw))
	switch v {
	case "json":
		return "json"
	case "text":
		return "text"
	case "true", "1", "yes":
		return "text"
	case "", "false", "0", "no":
		return ""
	default:
		fmt.Fprintf(os.Stderr, "structcli: unrecognized debug format %q, falling back to text\n", raw)
		return "text"
	}
}

// GetFormat returns the active debug format ("text" or "json") for the
// command, or "" if debug is not active.
func GetFormat(c *cobra.Command) string {
	debugFlagName := "debug-options"
	if currentFlagName, ok := c.Annotations[FlagAnnotation]; ok {
		debugFlagName = currentFlagName
	}

	rootC := c.Root()

	// Check the flag directly first.
	if debugFlag := rootC.PersistentFlags().Lookup(debugFlagName); debugFlag != nil && debugFlag.Changed {
		return normalizeFormat(debugFlag.Value.String())
	}

	// Check viper for other sources (env var, config).
	rootS := internalscope.Get(rootC)
	rootV := rootS.Viper()
	if raw := rootV.GetString(debugFlagName); raw != "" {
		return normalizeFormat(raw)
	}

	return ""
}

// IsDebugActive checks if the debug option is set for the command c,
// either through a command-line flag or an environment variable.
func IsDebugActive(c *cobra.Command) bool {
	return GetFormat(c) != ""
}
