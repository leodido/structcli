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

// LLMsTxtOptions configures the llms.txt generator.
type LLMsTxtOptions struct {
	ModulePath string // Go module path (eg. "github.com/myorg/mycli") — used to derive project URL
	IncludeMCP bool   // Mention --mcp in the output
}

// LLMsTxt generates an llms.txt file from a cobra command tree.
// Returns the file content as bytes.
func LLMsTxt(rootCmd *cobra.Command, opts LLMsTxtOptions) ([]byte, error) {
	schemas, err := structcli.JSONSchema(rootCmd, jsonschema.WithFullTree())
	if err != nil {
		return nil, fmt.Errorf("failed to get command schemas: %w", err)
	}

	if len(schemas) == 0 {
		return nil, fmt.Errorf("no command schemas found")
	}

	var buf bytes.Buffer

	rootSchema := schemas[0]
	cliName := rootSchema.Name

	// H1 heading with CLI name
	fmt.Fprintf(&buf, "# %s\n", cliName)
	if opts.ModulePath != "" {
		fmt.Fprintf(&buf, "\nhttps://%s\n", opts.ModulePath)
	}

	// Blockquote summary
	if rootSchema.Description != "" {
		fmt.Fprintf(&buf, "\n> %s\n", rootSchema.Description)
	}

	// Collect leaf commands (commands with flags and no subcommands, or commands with RunE)
	type leafCommand struct {
		schema      *structcli.CommandSchema
		cmd         *cobra.Command
		commandPath string
	}
	var leaves []leafCommand

	// Sort schemas by command path
	sortedSchemas := make([]*structcli.CommandSchema, len(schemas))
	copy(sortedSchemas, schemas)
	sort.Slice(sortedSchemas, func(i, j int) bool {
		return sortedSchemas[i].CommandPath < sortedSchemas[j].CommandPath
	})

	// Build a map of command path -> cobra.Command for RunE detection
	cmdMap := buildCommandMap(rootCmd)

	for _, s := range sortedSchemas {
		cmd := cmdMap[s.CommandPath]
		isLeaf := len(s.Subcommands) == 0 && (len(s.Flags) > 0 || (cmd != nil && cmd.RunE != nil))
		if isLeaf {
			leaves = append(leaves, leafCommand{schema: s, cmd: cmd, commandPath: s.CommandPath})
		}
	}

	// Commands index section
	if len(leaves) > 0 {
		fmt.Fprintf(&buf, "\n## Commands\n\n")
		for _, leaf := range leaves {
			anchor := toAnchor(leaf.schema.Name)
			description := leaf.schema.Description
			if description == "" {
				description = leaf.schema.Name
			}
			fmt.Fprintf(&buf, "- [%s](#%s): %s\n", leaf.commandPath, anchor, description)
		}
	}

	// Per-command sections
	for _, leaf := range leaves {
		fmt.Fprintf(&buf, "\n## %s\n", leaf.schema.Name)

		if leaf.schema.Description != "" {
			fmt.Fprintf(&buf, "\n%s\n", leaf.schema.Description)
		}

		// Flags section (sorted alphabetically)
		if len(leaf.schema.Flags) > 0 {
			fmt.Fprintf(&buf, "\n### Flags\n\n")
			flagNames := make([]string, 0, len(leaf.schema.Flags))
			for name := range leaf.schema.Flags {
				flagNames = append(flagNames, name)
			}
			sort.Strings(flagNames)

			for _, name := range flagNames {
				f := leaf.schema.Flags[name]
				defaultStr := ""
				if f.Default != "" {
					defaultStr = fmt.Sprintf(", default: %s", f.Default)
				}
				fmt.Fprintf(&buf, "- `--%s` (%s%s): %s\n", f.Name, f.Type, defaultStr, f.Description)
			}
		}

		// Environment Variables section
		var envEntries []struct {
			envVar   string
			flagName string
		}
		flagNames := make([]string, 0, len(leaf.schema.Flags))
		for name := range leaf.schema.Flags {
			flagNames = append(flagNames, name)
		}
		sort.Strings(flagNames)
		for _, name := range flagNames {
			f := leaf.schema.Flags[name]
			for _, envVar := range f.EnvVars {
				envEntries = append(envEntries, struct {
					envVar   string
					flagName string
				}{envVar: envVar, flagName: f.Name})
			}
		}

		if len(envEntries) > 0 {
			fmt.Fprintf(&buf, "\n### Environment Variables\n\n")
			for _, entry := range envEntries {
				fmt.Fprintf(&buf, "- `%s`: maps to `--%s`\n", entry.envVar, entry.flagName)
			}
		}
	}

	// Optional section
	var optionalItems []string
	optionalItems = append(optionalItems, fmt.Sprintf("- [JSON Schema](#json-schema): Machine-readable schema available via `%s --jsonschema`", cliName))
	if opts.IncludeMCP {
		optionalItems = append(optionalItems, fmt.Sprintf("- [MCP Server](#mcp): Available as MCP server via `%s --mcp`", cliName))
	}

	if len(optionalItems) > 0 {
		fmt.Fprintf(&buf, "\n## Optional\n\n")
		for _, item := range optionalItems {
			fmt.Fprintf(&buf, "%s\n", item)
		}
	}

	return buf.Bytes(), nil
}

// toAnchor converts a command name to a kebab-case anchor.
// For example, "usr add" becomes "usr-add", "srv" becomes "srv".
func toAnchor(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), " ", "-")
}

