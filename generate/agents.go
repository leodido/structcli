package generate

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/jsonschema"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// flagKitAnnotation mirrors [flagkit.FlagKitAnnotation] to avoid a dependency
// from the generate package on the flagkit package.
const flagKitAnnotation = "___leodido_structcli_flagkit"

// AgentsOptions configures the AGENTS.md generator.
type AgentsOptions struct {
	ModulePath string // Go module path for install instructions (eg. "github.com/myorg/mycli")
	IncludeMCP bool   // Mention --mcp in Machine Interface section
}

// Agents generates an AGENTS.md file from a cobra command tree.
// Returns the file content as bytes.
func Agents(rootCmd *cobra.Command, opts AgentsOptions) ([]byte, error) {
	schemas, err := structcli.JSONSchema(rootCmd, jsonschema.WithFullTree())
	if err != nil {
		return nil, fmt.Errorf("generating schemas: %w", err)
	}
	if len(schemas) == 0 {
		return nil, fmt.Errorf("no commands found")
	}

	// Sort for deterministic output
	schemas = sortedSchemas(schemas)

	root := schemas[0]
	cliName := root.Name

	// Filter to callable commands only (commands with a Run/RunE handler)
	cmdMap := buildCommandMap(rootCmd)
	var callables []*structcli.CommandSchema
	for _, s := range schemas {
		if isCallableCommand(s, cmdMap[s.CommandPath]) {
			callables = append(callables, s)
		}
	}

	var buf bytes.Buffer

	// Header
	fmt.Fprintf(&buf, "# %s\n\n", cliName)
	if root.Description != "" {
		fmt.Fprintf(&buf, "%s\n\n", root.Description)
	}

	// Installation
	if opts.ModulePath != "" {
		fmt.Fprintf(&buf, "## Installation\n\n```bash\ngo install %s@latest\n```\n\n", opts.ModulePath)
	}

	// Commands table
	fmt.Fprintf(&buf, "## Commands\n\n")
	fmt.Fprintf(&buf, "| Command | Description | Required Flags |\n")
	fmt.Fprintf(&buf, "|---------|-------------|---------------|\n")
	for _, s := range callables {
		reqFlags := requiredFlags(s)
		desc := s.Description
		if desc == "" {
			desc = "-"
		}
		fmt.Fprintf(&buf, "| `%s` | %s | %s |\n", s.CommandPath, desc, reqFlags)
	}
	buf.WriteString("\n")

	// Flags per command
	fmt.Fprintf(&buf, "## Configuration\n\n### Flags\n\n")
	for _, s := range callables {
		if len(s.Flags) == 0 {
			continue
		}
		fmt.Fprintf(&buf, "#### `%s`\n\n", s.CommandPath)
		fmt.Fprintf(&buf, "| Flag | Type | Default | Description |\n")
		fmt.Fprintf(&buf, "|------|------|---------|-------------|\n")
		for _, f := range sortedFlags(s) {
			if f.EnvOnly {
				continue
			}
			def := f.Default
			if def == "" {
				def = "-"
			}
			desc := f.Description
			if desc == "" {
				desc = "-"
			}
			if len(f.Enum) > 0 {
				desc += fmt.Sprintf(" (%s)", strings.Join(f.Enum, ", "))
			}
			fmt.Fprintf(&buf, "| `--%s` | %s | %s | %s |\n", f.Name, f.Type, def, desc)
		}
		buf.WriteString("\n")
	}

	// Environment variables (aggregated, deduplicated)
	envRows := collectEnvVars(callables)
	if len(envRows) > 0 {
		fmt.Fprintf(&buf, "### Environment Variables\n\n")
		fmt.Fprintf(&buf, "| Variable | Flag | Default |\n")
		fmt.Fprintf(&buf, "|----------|------|---------|\n")
		for _, e := range envRows {
			def := e.defVal
			if def == "" {
				def = "-"
			}
			flag := fmt.Sprintf("`--%s`", e.flagName)
			if e.envOnly {
				flag = "*(env only)*"
			}
			fmt.Fprintf(&buf, "| `%s` | %s | %s |\n", e.envVar, flag, def)
		}
		buf.WriteString("\n")
	}

	// Config File section — use annotation, not hardcoded flag name
	configFlagName := findConfigFlagName(rootCmd)
	if configFlagName != "" {
		fmt.Fprintf(&buf, "### Config File\n\nSupports YAML/JSON/TOML config files. Use `--%s` to specify path.\n\n", configFlagName)
	}

	// Machine Interface
	fmt.Fprintf(&buf, "## Machine Interface\n\n")
	fmt.Fprintf(&buf, "- JSON Schema: `%s --jsonschema`\n", cliName)
	fmt.Fprintf(&buf, "- Structured errors: JSON on stderr with semantic exit codes\n")
	if opts.IncludeMCP {
		fmt.Fprintf(&buf, "- MCP server: `%s --mcp`\n", cliName)
	}
	buf.WriteString("\n")

	// Development Notes — emitted when flagkit types are detected
	if hasFlagKitFlags(rootCmd) {
		buf.WriteString("## Development Notes\n\n")
		buf.WriteString("This CLI uses [structcli](https://github.com/leodido/structcli) with the `flagkit` package\n")
		buf.WriteString("for common flag patterns. When extending this CLI, prefer embedding `flagkit` types over\n")
		buf.WriteString("declaring ad-hoc flags for standard concerns (log level, output format, follow/streaming, etc.).\n\n")
		buf.WriteString("See `go doc github.com/leodido/structcli/flagkit` for available types.\n")
	}

	return buf.Bytes(), nil
}

type envRow struct {
	envVar   string
	flagName string
	defVal   string
	envOnly  bool
}

func collectEnvVars(schemas []*structcli.CommandSchema) []envRow {
	seen := map[string]bool{}
	var rows []envRow
	for _, s := range schemas {
		for _, f := range sortedFlags(s) {
			for _, ev := range f.EnvVars {
				if seen[ev] {
					continue
				}
				seen[ev] = true
				rows = append(rows, envRow{envVar: ev, flagName: f.Name, defVal: f.Default, envOnly: f.EnvOnly})
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].envVar < rows[j].envVar })
	return rows
}

func requiredFlags(s *structcli.CommandSchema) string {
	var req []string
	for _, f := range sortedFlags(s) {
		if f.Required && !f.EnvOnly {
			req = append(req, fmt.Sprintf("`--%s`", f.Name))
		}
	}
	if len(req) == 0 {
		return ""
	}
	return strings.Join(req, ", ")
}

// hasFlagKitFlags walks the command tree and returns true if any flag carries
// the flagkit annotation, indicating the CLI uses flagkit types.
func hasFlagKitFlags(root *cobra.Command) bool {
	found := false
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if found {
			return
		}
		c.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Annotations != nil {
				if _, ok := f.Annotations[flagKitAnnotation]; ok {
					found = true
				}
			}
		})
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	return found
}

// findConfigFlagName checks if the root command has a config flag registered
// by structcli's SetupConfig. Returns the flag name or empty string.
func findConfigFlagName(rootCmd *cobra.Command) string {
	// Check the annotation set by structcli.SetupConfig
	if rootCmd.Annotations != nil {
		if name, ok := rootCmd.Annotations["___leodido_structcli_configflagname"]; ok {
			return name
		}
	}
	// Fallback: check if a "config" flag exists as persistent
	if rootCmd.PersistentFlags().Lookup("config") != nil {
		return "config"
	}
	return ""
}
