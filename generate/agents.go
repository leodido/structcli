package generate

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/jsonschema"
	"github.com/spf13/cobra"
)

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

	// Sort schemas by command path for deterministic output.
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].CommandPath < schemas[j].CommandPath
	})

	root := schemas[0]
	cliName := root.Name

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
	for _, s := range schemas {
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
	for _, s := range schemas {
		if len(s.Flags) == 0 {
			continue
		}
		fmt.Fprintf(&buf, "#### `%s`\n\n", s.CommandPath)
		fmt.Fprintf(&buf, "| Flag | Type | Default | Description |\n")
		fmt.Fprintf(&buf, "|------|------|---------|-------------|\n")
		for _, f := range sortedFlags(s) {
			def := f.Default
			if def == "" {
				def = "-"
			}
			desc := f.Description
			if desc == "" {
				desc = "-"
			}
			fmt.Fprintf(&buf, "| `--%s` | %s | %s | %s |\n", f.Name, f.Type, def, desc)
		}
		buf.WriteString("\n")
	}

	// Environment variables (aggregated)
	envRows := collectEnvVars(schemas)
	if len(envRows) > 0 {
		fmt.Fprintf(&buf, "### Environment Variables\n\n")
		fmt.Fprintf(&buf, "| Variable | Flag | Default |\n")
		fmt.Fprintf(&buf, "|----------|------|---------|\n")
		for _, e := range envRows {
			def := e.defVal
			if def == "" {
				def = "-"
			}
			fmt.Fprintf(&buf, "| `%s` | `--%s` | %s |\n", e.envVar, e.flagName, def)
		}
		buf.WriteString("\n")
	}

	// Config File section
	if rootCmd.Flags().Lookup("config") != nil {
		fmt.Fprintf(&buf, "### Config File\n\nSupports YAML/JSON/TOML config files. Use `--config` to specify path.\n\n")
	}

	// Machine Interface
	fmt.Fprintf(&buf, "## Machine Interface\n\n")
	fmt.Fprintf(&buf, "- JSON Schema: `%s --jsonschema` (per-command schema)\n", cliName)
	fmt.Fprintf(&buf, "- Structured errors: JSON on stderr with semantic exit codes\n")
	if opts.IncludeMCP {
		fmt.Fprintf(&buf, "- MCP server: `%s --mcp`\n", cliName)
	}

	return buf.Bytes(), nil
}

type envRow struct {
	envVar   string
	flagName string
	defVal   string
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
				rows = append(rows, envRow{envVar: ev, flagName: f.Name, defVal: f.Default})
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].envVar < rows[j].envVar })
	return rows
}

func requiredFlags(s *structcli.CommandSchema) string {
	var req []string
	for _, f := range sortedFlags(s) {
		if f.Required {
			req = append(req, fmt.Sprintf("`--%s`", f.Name))
		}
	}
	if len(req) == 0 {
		return ""
	}
	return strings.Join(req, ", ")
}
