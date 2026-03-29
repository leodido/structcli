// Package generate produces static discovery files from structcli command trees.
//
// All generators consume [structcli.JSONSchema] with [jsonschema.WithFullTree]
// and produce []byte output. The caller decides where to write the files.
//
// Supported formats:
//   - [Skill]: SKILL.md for Claude.ai, Claude Code, Claude API
//   - [LLMsTxt]: llms.txt for any LLM (emerging web standard)
//   - [Agents]: AGENTS.md for coding agents (Linux Foundation standard)
package generate

import (
	"sort"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
)

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
