package generate

import (
	"bytes"
	"fmt"
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

	// Sort for deterministic output
	schemas = sortedSchemas(schemas)

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

	// Collect directly invokable commands using shared logic.
	type callableCommand struct {
		schema *structcli.CommandSchema
		cmd    *cobra.Command
	}
	var callables []callableCommand

	cmdMap := buildCommandMap(rootCmd)

	for _, s := range schemas {
		cmd := cmdMap[s.CommandPath]
		if isCallableCommand(s, cmd) {
			callables = append(callables, callableCommand{schema: s, cmd: cmd})
		}
	}

	// Commands index section — use CommandPath for unique anchors
	if len(callables) > 0 {
		fmt.Fprintf(&buf, "\n## Commands\n\n")
		for _, callable := range callables {
			anchor := toKebab(callable.schema.CommandPath)
			description := callable.schema.Description
			if description == "" {
				description = "-"
			}
			fmt.Fprintf(&buf, "- [%s](#%s): %s\n", callable.schema.CommandPath, anchor, description)
		}
	}

	// Per-command sections
	for _, callable := range callables {
		// Use CommandPath as heading for uniqueness
		fmt.Fprintf(&buf, "\n## %s\n", callable.schema.CommandPath)

		if callable.schema.Description != "" {
			fmt.Fprintf(&buf, "\n%s\n", callable.schema.Description)
		}

		// Build sorted flags once for this command
		flagNames := sortedFlagNames(callable.schema.Flags)

		// Flags section (excludes env-only fields)
		hasFlags := false
		for _, name := range flagNames {
			if !callable.schema.Flags[name].EnvOnly {
				hasFlags = true

				break
			}
		}
		if hasFlags {
			fmt.Fprintf(&buf, "\n### Flags\n\n")

			for _, name := range flagNames {
				f := callable.schema.Flags[name]
				if f.EnvOnly {
					continue
				}
				parts := []string{f.Type}
				if f.Default != "" {
					parts = append(parts, fmt.Sprintf("default: %s", f.Default))
				}
				if f.Required {
					parts = append(parts, "required")
				}
				descr := f.Description
				if descr == "" {
					descr = "-"
				}
				fmt.Fprintf(&buf, "- `--%s` (%s): %s\n", f.Name, strings.Join(parts, ", "), descr)
			}
		}

		// Environment Variables section
		var envEntries []struct {
			envVar   string
			flagName string
			envOnly  bool
		}
		for _, name := range flagNames {
			f := callable.schema.Flags[name]
			for _, envVar := range f.EnvVars {
				envEntries = append(envEntries, struct {
					envVar   string
					flagName string
					envOnly  bool
				}{envVar: envVar, flagName: f.Name, envOnly: f.EnvOnly})
			}
		}

		if len(envEntries) > 0 {
			fmt.Fprintf(&buf, "\n### Environment Variables\n\n")
			for _, entry := range envEntries {
				if entry.envOnly {
					fmt.Fprintf(&buf, "- `%s`: env only (no CLI flag)\n", entry.envVar)
				} else {
					fmt.Fprintf(&buf, "- `%s`: maps to `--%s`\n", entry.envVar, entry.flagName)
				}
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
