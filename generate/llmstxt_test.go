package generate_test

import (
	"strings"
	"testing"

	"github.com/leodido/structcli/generate"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMsTxt_H1Heading(t *testing.T) {
	root := buildTestTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(string(out), "# myapp\n"))
}

func TestLLMsTxt_ModulePath(t *testing.T) {
	root := buildTestTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{ModulePath: "github.com/test/myapp"})
	require.NoError(t, err)

	assert.Contains(t, string(out), "https://github.com/test/myapp")
}

func TestLLMsTxt_NoURLWithoutModulePath(t *testing.T) {
	root := buildTestTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)

	assert.NotContains(t, string(out), "https://")
}

func TestLLMsTxt_Blockquote(t *testing.T) {
	root := buildTestTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)

	assert.Contains(t, string(out), "> A test CLI application")
}

func TestLLMsTxt_CommandsIndex(t *testing.T) {
	root := buildTestTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "## Commands")
	assert.Contains(t, content, "myapp serve")
	assert.Contains(t, content, "myapp config")
}

func TestLLMsTxt_PerCommandFlags(t *testing.T) {
	root := buildTestTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)

	content := string(out)
	// Flags should be listed with type and default
	assert.Contains(t, content, "--port")
	assert.Contains(t, content, "--host")
	assert.Contains(t, content, "default: 3000")
	assert.Contains(t, content, "default: localhost")
}

func TestLLMsTxt_EnvVars(t *testing.T) {
	root := buildTestTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "### Environment Variables")
	assert.Contains(t, content, "SERVE_PORT")
	assert.Contains(t, content, "SERVE_HOST")
}

func TestLLMsTxt_OptionalSection(t *testing.T) {
	root := buildTestTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "## Optional")
	assert.Contains(t, content, "--jsonschema")
}

func TestLLMsTxt_IncludeMCP(t *testing.T) {
	root := buildTestTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{IncludeMCP: true})
	require.NoError(t, err)

	assert.Contains(t, string(out), "--mcp")
}

func TestLLMsTxt_NoMCPByDefault(t *testing.T) {
	root := buildTestTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)

	assert.NotContains(t, string(out), "--mcp")
}

func TestLLMsTxt_FlagsSorted(t *testing.T) {
	root := buildTestTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)

	content := string(out)
	hostIdx := strings.Index(content, "--host")
	portIdx := strings.Index(content, "--port")
	require.Greater(t, hostIdx, 0, "should contain --host")
	require.Greater(t, portIdx, 0, "should contain --port")
	assert.Less(t, hostIdx, portIdx, "host should come before port alphabetically")
}

func TestLLMsTxt_RequiredFlagMarked(t *testing.T) {
	root := buildTestTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)

	content := string(out)
	// Port is required — should say so
	assert.Contains(t, content, "required")
}

func TestLLMsTxt_DuplicateNamesUniqueAnchors(t *testing.T) {
	root := buildDuplicateNameTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)

	content := string(out)
	// Should use CommandPath-based anchors, not Name-based
	assert.Contains(t, content, "#mycli-db-add")
	assert.Contains(t, content, "#mycli-user-add")
	// Should NOT have duplicate simple anchors
	assert.NotContains(t, content, "(#add)")
}

func TestLLMsTxt_MinimalCLI(t *testing.T) {
	root := buildMinimalTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "# bare")
}

func TestLLMsTxt_EmptyDescription(t *testing.T) {
	root := buildNoDescriptionTree()
	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)

	content := string(out)
	// Should not crash, no blockquote for empty description
	assert.NotContains(t, content, "> \n")
}

func TestLLMsTxt_EmptyFlagDescription(t *testing.T) {
	noop := func(cmd *cobra.Command, args []string) error { return nil }
	root := &cobra.Command{Use: "app", Short: "App", RunE: noop}
	root.Flags().String("silent", "", "")

	out, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)

	content := string(out)
	// Empty flag description should show "-"
	assert.Contains(t, content, "--silent")
	assert.Contains(t, content, ": -")
}
