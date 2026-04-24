package structcli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/leodido/structcli/debug"
	internalcmd "github.com/leodido/structcli/internal/cmd"
	internaldebug "github.com/leodido/structcli/internal/debug"
	internalenv "github.com/leodido/structcli/internal/env"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// SetupDebug creates the --debug-options global flag and sets up debug behavior.
//
// Works only for the root command.
func SetupDebug(rootC *cobra.Command, debugOpts debug.Options) error {
	if rootC.Parent() != nil {
		return fmt.Errorf("SetupDebug must be called on the root command")
	}

	// Determine app name from root command
	appName := GetOrSetAppName(debugOpts.AppName, rootC.Name())
	if appName == "" {
		return fmt.Errorf("couldn't determine the app name")
	}

	// Compute flag and environment variable names
	flagName := debugOpts.FlagName
	if flagName == "" {
		flagName = "debug-options"
	}
	envvName := internalenv.NormEnv(debugOpts.EnvVar)
	if debugOpts.EnvVar == "" {
		normFlagName := internalenv.NormEnv(flagName)
		if currentPrefix := EnvPrefix(); currentPrefix != "" {
			envvName = fmt.Sprintf("%s_%s", currentPrefix, normFlagName)
		} else {
			envvName = fmt.Sprintf("%s_%s", internalenv.NormEnv(appName), normFlagName)
		}
	}

	// Store the actual debug options flag name in the root command annotations
	if rootC.Annotations == nil {
		rootC.Annotations = make(map[string]string)
	}
	rootC.Annotations[internaldebug.FlagAnnotation] = flagName

	// Add persistent flag to root command.
	// NoOptDefVal makes bare --debug-options (no value) default to "text",
	// preserving backward compatibility with the old bool flag.
	rootC.PersistentFlags().String(flagName, "", "debug output format (text, json)")
	rootC.PersistentFlags().Lookup(flagName).NoOptDefVal = "text"

	// Add environment annotation
	mustSetAnnotation(rootC.PersistentFlags(), flagName, internalenv.FlagAnnotation, []string{envvName})

	// Ensure environment binding happens
	cobra.OnInitialize(func() {
		if err := internalenv.BindEnv(rootC); err != nil {
			fmt.Fprintf(os.Stderr, "structcli: debug env binding error: %v\n", err)
		}
	})

	// Wrap all commands run hooks
	if debugOpts.Exit {
		// Ensure the root command is runnable so cobra calls PreRunE
		// instead of short-circuiting to Help().
		internalcmd.EnsureRunnable(rootC)

		// Wrap already-registered commands now.
		internalcmd.RecursivelyWrapRun(rootC)

		// Also wrap right before execution so commands added after SetupDebug are covered.
		cobra.OnInitialize(func() {
			internalcmd.RecursivelyWrapRun(rootC)
		})
	}

	// Regenerate usage templates for any commands already processed by Define()
	SetupUsage(rootC)

	return nil
}

// IsDebugActive checks if the debug option is set for the command c, either through a command-line flag or an environment variable.
func IsDebugActive(c *cobra.Command) bool {
	return internaldebug.IsDebugActive(c)
}

// UseDebug manually triggers debug output for the given options.
//
// When --debug-options=json, output goes to w as a JSON object.
// When --debug-options or --debug-options=text, output goes to w
// as a human-readable table.
//
// Debug output is automatically triggered when the debug flag is enabled.
func UseDebug(c *cobra.Command, w io.Writer) {
	format := internaldebug.GetFormat(c)
	if format == "" {
		return
	}

	v := GetViper(c)
	configV := GetConfigViper(c)

	switch format {
	case "json":
		writeDebugJSON(c, v, configV, w)
	default:
		writeDebugText(c, v, configV, w)
	}
}

// debugFlagState represents a single flag's resolved state.
type debugFlagState struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Default string `json:"default"`
	Changed bool   `json:"changed"`
	Source  string `json:"source"`
}

// debugOutput is the top-level JSON structure for debug output.
type debugOutput struct {
	Command string            `json:"command"`
	Flags   []debugFlagState  `json:"flags"`
	Values  map[string]any    `json:"values"`
}

func collectFlagStates(c *cobra.Command, configV *viper.Viper) []debugFlagState {
	var states []debugFlagState

	c.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		source := internaldebug.ResolveFlagSource(f, configV)
		states = append(states, debugFlagState{
			Name:    f.Name,
			Value:   f.Value.String(),
			Default: f.DefValue,
			Changed: f.Changed,
			Source:  string(source),
		})
	})

	// Sort by name for deterministic output.
	sort.Slice(states, func(i, j int) bool {
		return states[i].Name < states[j].Name
	})

	return states
}

func writeDebugJSON(c *cobra.Command, v *viper.Viper, configV *viper.Viper, w io.Writer) {
	out := debugOutput{
		Command: c.CommandPath(),
		Flags:   collectFlagStates(c, configV),
		Values:  v.AllSettings(),
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func writeDebugText(c *cobra.Command, v *viper.Viper, configV *viper.Viper, w io.Writer) {
	fmt.Fprintf(w, "Command: %s\n\n", c.CommandPath())

	states := collectFlagStates(c, configV)

	if len(states) > 0 {
		// Compute column widths for alignment.
		maxName, maxVal := 0, 0
		for _, s := range states {
			flagStr := "--" + s.Name
			if len(flagStr) > maxName {
				maxName = len(flagStr)
			}
			if len(s.Value) > maxVal {
				maxVal = len(s.Value)
			}
		}

		fmt.Fprintln(w, "Flags:")
		for _, s := range states {
			flagStr := "--" + s.Name
			sourceStr := formatSource(s, c)
			fmt.Fprintf(w, "  %-*s  %-*s  (%s)\n", maxName, flagStr, maxVal, s.Value, sourceStr)
		}
		fmt.Fprintln(w)
	}

	settings := v.AllSettings()
	if len(settings) > 0 {
		keys := make([]string, 0, len(settings))
		for k := range settings {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		fmt.Fprintln(w, "Values:")
		for _, k := range keys {
			fmt.Fprintf(w, "  %s: %v\n", k, settings[k])
		}
	}
}

// formatSource returns a human-readable source label for text output.
// For env sources, it includes the env var name.
func formatSource(s debugFlagState, c *cobra.Command) string {
	if s.Source != "env" {
		return s.Source
	}

	// Look up the env var name from the flag annotation.
	if f := c.Flags().Lookup(s.Name); f != nil {
		if envs, ok := f.Annotations[internalenv.FlagAnnotation]; ok && len(envs) > 0 {
			return "env: " + strings.Join(envs, ", ")
		}
	}

	return "env"
}
