package generate_test

import (
	"strings"
	"testing"

	"github.com/leodido/structcli/generate"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgents_BasicOutput(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "# myapp")
	assert.Contains(t, content, "A test CLI application")
}

func TestAgents_CommandsTable(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "## Commands")
	assert.Contains(t, content, "| Command | Description | Required Flags |")
	assert.Contains(t, content, "myapp serve")
	assert.Contains(t, content, "myapp config")
}

func TestAgents_RequiredFlags(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	content := string(out)
	// The serve command has port as required
	serveLine := ""
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, "myapp serve") && strings.HasPrefix(line, "|") {
			serveLine = line
			break
		}
	}
	require.NotEmpty(t, serveLine, "should find serve in commands table")
	assert.Contains(t, serveLine, "--port")
}

func TestAgents_FlagsTableSorted(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	content := string(out)
	// Find the serve flags section and check host comes before port
	serveSection := extractSection(content, "#### `myapp serve`")
	require.NotEmpty(t, serveSection)

	hostIdx := strings.Index(serveSection, "--host")
	portIdx := strings.Index(serveSection, "--port")
	require.Greater(t, hostIdx, 0)
	require.Greater(t, portIdx, 0)
	assert.Less(t, hostIdx, portIdx, "host should come before port alphabetically")
}

func TestAgents_EnvVarsTable(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "### Environment Variables")
	assert.Contains(t, content, "| Variable | Flag | Default |")
	assert.Contains(t, content, "SERVE_PORT")
	assert.Contains(t, content, "SERVE_HOST")
}

func TestAgents_Installation(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Agents(root, generate.AgentsOptions{ModulePath: "github.com/myorg/myapp"})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "## Installation")
	assert.Contains(t, content, "go install github.com/myorg/myapp@latest")
}

func TestAgents_InstallationOmittedWithoutModulePath(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	assert.NotContains(t, string(out), "## Installation")
}

func TestAgents_MachineInterface(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "## Machine Interface")
	assert.Contains(t, content, "--jsonschema")
}

func TestAgents_IncludeMCP(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Agents(root, generate.AgentsOptions{IncludeMCP: true})
	require.NoError(t, err)

	assert.Contains(t, string(out), "--mcp")
}

func TestAgents_MinimalCLI(t *testing.T) {
	root := buildMinimalTree()
	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "# bare")
	assert.Contains(t, content, "| `bare` |")
}

func TestAgents_EmptyDescription(t *testing.T) {
	root := buildNoDescriptionTree()
	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	content := string(out)
	// Empty description should show "-" in table
	assert.Contains(t, content, "| `nodesc` | - |")
}

func TestAgents_EmptyFlagDescription(t *testing.T) {
	noop := func(cmd *cobra.Command, args []string) error { return nil }
	root := &cobra.Command{Use: "app", Short: "App", RunE: noop}
	root.Flags().String("silent", "", "")

	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	content := string(out)
	// Empty flag description should show "-"
	line := extractSection(content, "| `--silent`")
	assert.Contains(t, line, "| - |")
}

func TestAgents_NonCallableCommandsExcluded(t *testing.T) {
	// buildRunnableParentTree: root (non-callable), srv (callable, Run), srv version (callable, RunE)
	root := buildRunnableParentTree()
	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	content := string(out)

	// Non-callable root should not appear in the commands table
	assert.NotContains(t, content, "| `myapp` |", "non-callable root should not appear in commands table")

	// Callable subcommands should appear
	assert.Contains(t, content, "| `myapp srv` |")
	assert.Contains(t, content, "| `myapp srv version` |")
}

func TestAgents_NonCallableCommandFlagsExcluded(t *testing.T) {
	noop := func(cmd *cobra.Command, args []string) error { return nil }

	// Root is non-callable container with a persistent flag
	root := &cobra.Command{Use: "app", Short: "A CLI"}
	root.PersistentFlags().Bool("verbose", false, "Enable verbose output")

	sub := &cobra.Command{Use: "run", Short: "Run something", RunE: noop}
	sub.Flags().Int("count", 1, "Repeat count")
	root.AddCommand(sub)

	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	content := string(out)

	// Non-callable root's flags section should not appear
	assert.NotContains(t, content, "#### `app`", "non-callable root should not have a flags section")

	// Callable sub's flags should appear
	assert.Contains(t, content, "#### `app run`")
	assert.Contains(t, content, "--count")
}

func TestAgents_ZeroFlagCommand(t *testing.T) {
	noop := func(cmd *cobra.Command, args []string) error { return nil }
	root := &cobra.Command{Use: "app", Short: "A CLI", RunE: noop}

	// Subcommand with no flags
	sub := &cobra.Command{Use: "ping", Short: "Ping the server", RunE: noop}
	root.AddCommand(sub)

	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	content := string(out)

	// Both commands should appear in the table
	assert.Contains(t, content, "| `app` |")
	assert.Contains(t, content, "| `app ping` |")

	// No flags section for ping (zero flags)
	assert.NotContains(t, content, "#### `app ping`")
}

func TestAgents_FlagKitDevNotes(t *testing.T) {
	noop := func(cmd *cobra.Command, args []string) error { return nil }
	root := &cobra.Command{Use: "app", Short: "A CLI", RunE: noop}
	root.Flags().Bool("follow", false, "Stream output continuously")
	// Simulate flagkit annotation
	require.NoError(t, root.Flags().SetAnnotation("follow", "___leodido_structcli_flagkit", []string{"true"}))

	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "## Development Notes")
	assert.Contains(t, content, "flagkit")
	assert.Contains(t, content, "go doc github.com/leodido/structcli/flagkit")
}

func TestAgents_NoFlagKitDevNotes(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	assert.NotContains(t, string(out), "## Development Notes")
}

func TestAgents_FlagKitDevNotesOnSubcommand(t *testing.T) {
	noop := func(cmd *cobra.Command, args []string) error { return nil }
	root := &cobra.Command{Use: "app", Short: "A CLI"}
	sub := &cobra.Command{Use: "logs", Short: "Show logs", RunE: noop}
	sub.Flags().Bool("follow", false, "Stream output continuously")
	require.NoError(t, sub.Flags().SetAnnotation("follow", "___leodido_structcli_flagkit", []string{"true"}))
	root.AddCommand(sub)

	out, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)

	assert.Contains(t, string(out), "## Development Notes")
}

// --- helpers ---

func extractSection(content, heading string) string {
	idx := strings.Index(content, heading)
	if idx < 0 {
		return ""
	}
	rest := content[idx+len(heading):]
	// Find next heading of same or higher level
	nextIdx := strings.Index(rest, "\n#### ")
	if nextIdx < 0 {
		nextIdx = strings.Index(rest, "\n### ")
	}
	if nextIdx < 0 {
		nextIdx = strings.Index(rest, "\n## ")
	}
	if nextIdx < 0 {
		return rest
	}
	return rest[:nextIdx]
}
