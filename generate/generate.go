// Package generate produces static discovery files from structcli command trees.
//
// All generators consume [structcli.JSONSchema] with [jsonschema.WithFullTree]
// and produce []byte output. The caller decides where to write the files.
//
// The generators produce mechanically correct scaffolds — every flag name, type,
// default, env var, and required marker comes from the same struct definition that
// powers --jsonschema. Humans should add on top:
//   - Trigger phrases for skill discovery ("use when user asks to deploy")
//   - Workflow guidance and step-by-step instructions
//   - Realistic examples with domain-specific values
//   - Error handling advice and troubleshooting sections
//   - Negative triggers ("do NOT use for general file management")
//
// Supported formats:
//   - [Skill]: SKILL.md for Claude.ai, Claude Code, Claude API
//   - [LLMsTxt]: llms.txt for any LLM (emerging web standard)
//   - [Agents]: AGENTS.md for coding agents (Linux Foundation standard)
package generate

import (
	"sort"
	"strings"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
)

// maxSkillDescriptionLen is the maximum character length for the SKILL.md
// description field, per the Anthropic skill specification.
const maxSkillDescriptionLen = 1024

// buildCommandMap walks the cobra command tree and returns a map of
// command path → *cobra.Command for accessing fields not in CommandSchema
// (eg. Example, RunE).
func buildCommandMap(root *cobra.Command) map[string]*cobra.Command {
	m := make(map[string]*cobra.Command)
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		m[c.CommandPath()] = c
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	return m
}

// isLeafCommand returns true if a command is a leaf (callable, not just a group).
// A leaf command has no subcommands and either has flags or has a RunE handler.
// This is the single source of truth used by all generators.
func isLeafCommand(s *structcli.CommandSchema, cmd *cobra.Command) bool {
	return len(s.Subcommands) == 0 && (len(s.Flags) > 0 || (cmd != nil && cmd.RunE != nil))
}

// sortedSchemas returns a copy of schemas sorted by CommandPath for deterministic output.
func sortedSchemas(schemas []*structcli.CommandSchema) []*structcli.CommandSchema {
	sorted := make([]*structcli.CommandSchema, len(schemas))
	copy(sorted, schemas)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CommandPath < sorted[j].CommandPath
	})
	return sorted
}

// sortedFlagNames returns flag names from a schema sorted alphabetically.
func sortedFlagNames(flags map[string]*structcli.FlagSchema) []string {
	names := make([]string, 0, len(flags))
	for name := range flags {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// sortedFlags returns flags from a schema sorted alphabetically by name.
func sortedFlags(s *structcli.CommandSchema) []*structcli.FlagSchema {
	names := sortedFlagNames(s.Flags)
	flags := make([]*structcli.FlagSchema, 0, len(names))
	for _, n := range names {
		flags = append(flags, s.Flags[n])
	}
	return flags
}

// toKebab converts a string to kebab-case (lowercase, spaces to hyphens).
// Used for SKILL.md names and markdown anchors.
func toKebab(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), " ", "-")
}

// yamlQuote returns a YAML-safe representation of a string value.
// Strings containing YAML special characters that would break parsing
// are wrapped in double quotes.
func yamlQuote(s string) string {
	// Characters that require quoting in YAML scalar values
	if strings.ContainsAny(s, ":{}[]&*#?|<>=!%@`,'\"\\") {
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		return `"` + escaped + `"`
	}
	return s
}
