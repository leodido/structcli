package generate

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/jsonschema"
	"github.com/spf13/cobra"
)

// SkillOptions configures the SKILL.md generator.
type SkillOptions struct {
	Name      string // Override skill name (default: root command name, kebab-case)
	Author    string // metadata.author (optional)
	Version   string // metadata.version (optional)
	MCPServer string // metadata.mcp-server (optional)
}

// Skill generates a SKILL.md file from a cobra command tree.
// Returns the file content as bytes.
func Skill(rootCmd *cobra.Command, opts SkillOptions) ([]byte, error) {
	schemas, err := structcli.JSONSchema(rootCmd, jsonschema.WithFullTree())
	if err != nil {
		return nil, fmt.Errorf("generating JSON schema tree: %w", err)
	}

	if len(schemas) == 0 {
		return nil, fmt.Errorf("no command schemas found")
	}

	// Build a map from command path to the cobra command for accessing Example, Aliases, etc.
	cmdMap := buildCommandMap(rootCmd)

	// Sort for deterministic output
	schemas = sortedSchemas(schemas)

	rootSchema := schemas[0]

	name := opts.Name
	if name == "" {
		name = toKebab(rootSchema.Name)
	}

	description := buildDescription(schemas, cmdMap)

	var buf bytes.Buffer

	// YAML frontmatter
	fmt.Fprintf(&buf, "---\n")
	fmt.Fprintf(&buf, "name: %s\n", name)
	fmt.Fprintf(&buf, "description: |\n")
	for _, line := range strings.Split(description, "\n") {
		fmt.Fprintf(&buf, "  %s\n", line)
	}

	if opts.Author != "" || opts.Version != "" || opts.MCPServer != "" {
		fmt.Fprintf(&buf, "metadata:\n")
		if opts.Author != "" {
			fmt.Fprintf(&buf, "  author: %s\n", yamlQuote(opts.Author))
		}
		if opts.Version != "" {
			fmt.Fprintf(&buf, "  version: %s\n", yamlQuote(opts.Version))
		}
		if opts.MCPServer != "" {
			fmt.Fprintf(&buf, "  mcp-server: %s\n", yamlQuote(opts.MCPServer))
		}
	}
	fmt.Fprintf(&buf, "---\n\n")

	// Body
	cliName := rootSchema.Name
	fmt.Fprintf(&buf, "# %s\n\n", cliName)
	fmt.Fprintf(&buf, "## Instructions\n\n")
	fmt.Fprintf(&buf, "### Available Commands\n")

	for _, schema := range schemas {
		cmd := cmdMap[schema.CommandPath]
		writeCommandSection(&buf, schema, cmd)
	}

	// Environment Variable Prefix section
	if rootSchema.EnvPrefix != "" {
		fmt.Fprintf(&buf, "\n### Environment Variable Prefix\n\n")
		fmt.Fprintf(&buf, "All environment variables use the `%s_` prefix.\n", rootSchema.EnvPrefix)
	}

	// Aggregate examples
	examples := collectExamples(schemas, cmdMap)
	if len(examples) > 0 {
		fmt.Fprintf(&buf, "\n### Examples\n")
		for _, ex := range examples {
			fmt.Fprintf(&buf, "\n#### %s\n\n", ex.commandPath)
			fmt.Fprintf(&buf, "```\n%s\n```\n", strings.TrimSpace(ex.example))
		}
	}

	return buf.Bytes(), nil
}

// commandExample pairs a command path with its example text.
type commandExample struct {
	commandPath string
	example     string
}

// collectExamples gathers examples from commands that have the Example field set.
func collectExamples(schemas []*structcli.CommandSchema, cmdMap map[string]*cobra.Command) []commandExample {
	var results []commandExample
	for _, schema := range schemas {
		cmd := cmdMap[schema.CommandPath]
		if cmd != nil && cmd.Example != "" {
			results = append(results, commandExample{
				commandPath: schema.CommandPath,
				example:     cmd.Example,
			})
		}
	}
	return results
}

// writeCommandSection writes the markdown section for a single command.
func writeCommandSection(buf *bytes.Buffer, schema *structcli.CommandSchema, cmd *cobra.Command) {
	fmt.Fprintf(buf, "\n#### `%s`\n\n", schema.CommandPath)

	if schema.Description != "" {
		fmt.Fprintf(buf, "%s\n", schema.Description)
	}

	// Flags table
	if len(schema.Flags) > 0 {
		writeFlagsTable(buf, schema.Flags)
	}

	// Env vars table
	envRows := collectEnvVarRows(schema.Flags)
	if len(envRows) > 0 {
		writeEnvVarsTable(buf, envRows)
	}

	// Per-command example
	if cmd != nil && cmd.Example != "" {
		fmt.Fprintf(buf, "\n**Example:**\n\n")
		fmt.Fprintf(buf, "```\n%s\n```\n", strings.TrimSpace(cmd.Example))
	}
}

// writeFlagsTable writes the flags markdown table.
func writeFlagsTable(buf *bytes.Buffer, flags map[string]*structcli.FlagSchema) {
	fmt.Fprintf(buf, "\n**Flags:**\n\n")
	fmt.Fprintf(buf, "| Flag | Type | Default | Required | Description |\n")
	fmt.Fprintf(buf, "|------|------|---------|----------|-------------|\n")

	names := sortedFlagNames(flags)
	for _, name := range names {
		f := flags[name]
		reqStr := "no"
		if f.Required {
			reqStr = "yes"
		}
		def := f.Default
		if def == "" {
			def = "-"
		}
		descr := f.Description
		if descr == "" {
			descr = "-"
		}
		fmt.Fprintf(buf, "| `--%s` | %s | %s | %s | %s |\n", f.Name, f.Type, def, reqStr, descr)
	}
}

// envVarRow represents a row in the environment variables table.
type envVarRow struct {
	variable string
	flag     string
	descr    string
}

// collectEnvVarRows collects env var rows from flags that have env vars.
func collectEnvVarRows(flags map[string]*structcli.FlagSchema) []envVarRow {
	var rows []envVarRow
	names := sortedFlagNames(flags)
	for _, name := range names {
		f := flags[name]
		for _, env := range f.EnvVars {
			rows = append(rows, envVarRow{
				variable: env,
				flag:     f.Name,
				descr:    f.Description,
			})
		}
	}
	return rows
}

// writeEnvVarsTable writes the environment variables markdown table.
func writeEnvVarsTable(buf *bytes.Buffer, rows []envVarRow) {
	fmt.Fprintf(buf, "\n**Environment Variables:**\n\n")
	fmt.Fprintf(buf, "| Variable | Flag | Description |\n")
	fmt.Fprintf(buf, "|----------|------|-------------|\n")
	for _, r := range rows {
		fmt.Fprintf(buf, "| `%s` | `--%s` | %s |\n", r.variable, r.flag, r.descr)
	}
}

// buildDescription generates the skill description from command schemas.
// It describes what the CLI does and when to use it (trigger phrases from leaf commands).
// The result is kept at or under [maxSkillDescriptionLen] characters and contains no XML tags,
// per the Anthropic skill specification.
func buildDescription(schemas []*structcli.CommandSchema, cmdMap map[string]*cobra.Command) string {
	if len(schemas) == 0 {
		return ""
	}

	root := schemas[0]
	var sb strings.Builder

	// What it does
	if root.Description != "" {
		sb.WriteString(root.Description)
		sb.WriteString(". ")
	}

	// Trigger phrases from leaf commands
	var triggers []string
	for _, schema := range schemas {
		cmd := cmdMap[schema.CommandPath]
		if isLeafCommand(schema, cmd) && schema.Description != "" {
			trigger := strings.ToLower(strings.TrimRight(schema.Description, "."))
			triggers = append(triggers, trigger)
		}
	}

	if len(triggers) > 0 {
		sb.WriteString("Use when you need to: ")
		sb.WriteString(strings.Join(triggers, ", "))
		sb.WriteString(".")
	}

	desc := sb.String()

	// Strip any XML tags (forbidden in SKILL.md frontmatter)
	desc = stripXMLTags(desc)

	// Truncate to maxSkillDescriptionLen
	if len(desc) > maxSkillDescriptionLen {
		desc = desc[:maxSkillDescriptionLen-3] + "..."
	}

	return desc
}

// stripXMLTags removes XML/HTML tags from a string.
func stripXMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}
