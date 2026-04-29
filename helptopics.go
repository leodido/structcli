package structcli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/leodido/structcli/helptopics"
	internalenv "github.com/leodido/structcli/internal/env"
	internalusage "github.com/leodido/structcli/internal/usage"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// SetupHelpTopics adds "env-vars" and "config-keys" reference commands to the
// root command. By default they appear as regular subcommands under "Available
// Commands:". Set ReferenceSection to move them into a dedicated "Reference:"
// section.
//
// Text is generated lazily at invocation time, so commands added after this
// call are included.
func SetupHelpTopics(rootC *cobra.Command, opts helptopics.Options) error {
	if rootC.Parent() != nil {
		return fmt.Errorf("SetupHelpTopics must be called on the root command")
	}

	if opts.ReferenceSection {
		if rootC.Annotations == nil {
			rootC.Annotations = map[string]string{}
		}
		rootC.Annotations[internalusage.HelpTopicReferenceSection] = "true"
	}

	envVarsCmd := &cobra.Command{
		Use:   "env-vars",
		Short: "List all environment variable bindings",
		Long:  "Show every flag-to-environment-variable mapping across all commands.",
		Annotations: map[string]string{
			internalusage.HelpTopicAnnotation: "true",
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprint(cmd.OutOrStdout(), buildEnvVarsTopic(rootC))

			return nil
		},
	}
	rootC.AddCommand(envVarsCmd)

	configKeysCmd := &cobra.Command{
		Use:   "config-keys",
		Short: "List all configuration file keys",
		Long:  "Show every valid configuration file key across all commands.",
		Annotations: map[string]string{
			internalusage.HelpTopicAnnotation: "true",
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprint(cmd.OutOrStdout(), buildConfigKeysTopic(rootC))

			return nil
		},
	}
	rootC.AddCommand(configKeysCmd)

	return nil
}

// IsHelpTopicCommand returns true if the command was registered by SetupHelpTopics.
func IsHelpTopicCommand(c *cobra.Command) bool {
	if c.Annotations == nil {
		return false
	}
	_, ok := c.Annotations[internalusage.HelpTopicAnnotation]

	return ok
}

// commandEnvBinding represents a single flag's env var binding.
type commandEnvBinding struct {
	envVars []string
	flag    string
	typ     string
	defVal  string
	envOnly bool
}

// commandConfigKey represents a single config key mapping.
type commandConfigKey struct {
	key      string
	flag     string
	typ      string
	defVal   string
	isAlias  bool
	aliasFor string
}

// buildEnvVarsTopic walks the command tree and generates the env-vars help text.
func buildEnvVarsTopic(rootC *cobra.Command) string {
	var b strings.Builder
	b.WriteString("Environment Variables\n")

	walkCommands(rootC, func(c *cobra.Command, path string) {
		bindings := collectEnvBindings(c)
		if len(bindings) == 0 {
			return
		}

		label := path
		if c == rootC {
			label = path + " (global)"
		}
		b.WriteString(fmt.Sprintf("\n  %s:\n", label))

		// Compute column widths.
		maxEnv, maxFlag := 0, 0
		for _, bind := range bindings {
			for _, env := range bind.envVars {
				if len(env) > maxEnv {
					maxEnv = len(env)
				}
			}
			flagStr := "--" + bind.flag
			if len(flagStr) > maxFlag {
				maxFlag = len(flagStr)
			}
		}

		for _, bind := range bindings {
			flagStr := "--" + bind.flag
			suffix := ""
			if bind.envOnly {
				suffix = "  (env-only)"
			}

			for i, env := range bind.envVars {
				if i == 0 {
					b.WriteString(fmt.Sprintf("    %-*s  %-*s  %-14s %s%s\n",
						maxEnv, env, maxFlag, flagStr, bind.typ, bind.defVal, suffix))
				} else {
					b.WriteString(fmt.Sprintf("    %-*s  (alias for %s)\n", maxEnv, env, bind.envVars[0]))
				}
			}
		}
	})

	return b.String()
}

// buildConfigKeysTopic walks the command tree and generates the config-keys help text.
func buildConfigKeysTopic(rootC *cobra.Command) string {
	var b strings.Builder
	b.WriteString("Configuration Keys\n")

	// Show config file locations if SetupConfig was called.
	if flagName, ok := rootC.Annotations[ConfigFlagAnnotation]; ok {
		if f := rootC.PersistentFlags().Lookup(flagName); f != nil {
			b.WriteString(fmt.Sprintf("\n  Config flag: --%s\n", flagName))
			if f.Usage != "" {
				b.WriteString(fmt.Sprintf("  %s\n", f.Usage))
			}
		}
	}

	walkCommands(rootC, func(c *cobra.Command, path string) {
		keys := collectConfigKeys(c)
		if len(keys) == 0 {
			return
		}

		label := path
		if c == rootC {
			label = path + " (global)"
		}
		b.WriteString(fmt.Sprintf("\n  %s:\n", label))

		// Compute column widths.
		maxKey, maxFlag := 0, 0
		for _, k := range keys {
			if len(k.key) > maxKey {
				maxKey = len(k.key)
			}
			if !k.isAlias {
				flagStr := "--" + k.flag
				if len(flagStr) > maxFlag {
					maxFlag = len(flagStr)
				}
			}
		}

		for _, k := range keys {
			if k.isAlias {
				b.WriteString(fmt.Sprintf("    %-*s  (alias for --%s)\n", maxKey, k.key, k.aliasFor))
			} else {
				flagStr := "--" + k.flag
				b.WriteString(fmt.Sprintf("    %-*s  %-*s  %-14s %s\n",
					maxKey, k.key, maxFlag, flagStr, k.typ, k.defVal))
			}
		}
	})

	b.WriteString("\n  Keys can be nested under the command name in the config file.\n")

	return b.String()
}

// walkCommands visits the root and all non-hidden subcommands depth-first.
// The root is visited with c.Name() (e.g. "mycli") while subcommands use
// c.CommandPath() (e.g. "mycli serve") so the root label stays short and
// subcommand labels show the full invocation path.
func walkCommands(c *cobra.Command, fn func(c *cobra.Command, path string)) {
	fn(c, c.Name())
	walkSubcommands(c, fn)
}

func walkSubcommands(parent *cobra.Command, fn func(c *cobra.Command, path string)) {
	for _, child := range parent.Commands() {
		if IsHelpTopicCommand(child) || child.IsAdditionalHelpTopicCommand() || !child.IsAvailableCommand() {
			continue
		}
		fn(child, child.CommandPath())
		walkSubcommands(child, fn)
	}
}

// collectEnvBindings extracts env var bindings from a command's own flags.
// Hidden flags are intentionally included: env-only flags (flagenv:"only") are
// hidden from --help but must appear here since env vars are their only input
// channel. They are marked with an (env-only) suffix.
func collectEnvBindings(c *cobra.Command) []commandEnvBinding {
	var bindings []commandEnvBinding

	visitLocalFlags(c, func(f *pflag.Flag) {
		envs, ok := f.Annotations[internalenv.FlagAnnotation]
		if !ok || len(envs) == 0 {
			return
		}

		_, envOnly := f.Annotations[internalenv.FlagEnvOnlyAnnotation]

		bindings = append(bindings, commandEnvBinding{
			envVars: envs,
			flag:    f.Name,
			typ:     f.Value.Type(),
			defVal:  formatDefault(f.DefValue),
			envOnly: envOnly,
		})
	})

	return bindings
}

// collectConfigKeys extracts config key mappings from a command's own flags.
// Hidden flags are excluded: env-only flags cannot be set via config files, and
// other manually hidden flags are intentionally kept out of the config reference.
func collectConfigKeys(c *cobra.Command) []commandConfigKey {
	var keys []commandConfigKey
	seen := map[string]bool{}

	visitLocalFlags(c, func(f *pflag.Flag) {
		if f.Hidden {
			return
		}

		flagName := f.Name
		typ := f.Value.Type()
		defVal := formatDefault(f.DefValue)

		// The flag name is always a valid config key.
		keys = append(keys, commandConfigKey{
			key:    flagName,
			flag:   flagName,
			typ:    typ,
			defVal: defVal,
		})
		seen[flagName] = true

		// The struct field path (lowercased) is an alias if it differs.
		if paths, ok := f.Annotations[flagPathAnnotation]; ok && len(paths) > 0 {
			alias := strings.ToLower(paths[0])
			if alias != flagName && !seen[alias] {
				keys = append(keys, commandConfigKey{
					key:      alias,
					isAlias:  true,
					aliasFor: flagName,
				})
				seen[alias] = true
			}
		}
	})

	// Sort: primary keys first (alphabetical), then aliases.
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].isAlias != keys[j].isAlias {
			return !keys[i].isAlias
		}
		return keys[i].key < keys[j].key
	})

	return keys
}

// visitLocalFlags visits flags that belong to this command (not inherited).
// For the root command this includes both regular and persistent flags
// (e.g. --config added by SetupConfig). For subcommands it excludes
// inherited persistent flags.
func visitLocalFlags(c *cobra.Command, fn func(*pflag.Flag)) {
	c.LocalFlags().VisitAll(fn)
}

// formatDefault formats a default value for display.
func formatDefault(v string) string {
	if v == "" {
		return `""`
	}

	return v
}
