package generate_test

import (
	"strings"
	"testing"

	"github.com/leodido/structcli/generate"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkill_YAMLFrontmatter(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.True(t, strings.HasPrefix(content, "---\n"), "should start with YAML frontmatter")

	secondDelim := strings.Index(content[4:], "---\n")
	require.Greater(t, secondDelim, 0, "should have closing frontmatter delimiter")

	assert.Contains(t, content, "name: myapp")
}

func TestSkill_NameKebabCase(t *testing.T) {
	// Cobra's Name() returns the first word of Use — so Use should already be kebab-case
	root := &cobra.Command{
		Use:  "my-cool-app",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}

	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	assert.Contains(t, string(out), "name: my-cool-app")
}

func TestSkill_NameOverride(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Skill(root, generate.SkillOptions{Name: "custom-skill-name"})
	require.NoError(t, err)

	assert.Contains(t, string(out), "name: custom-skill-name")
}

func TestSkill_DescriptionUnder1024(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	content := string(out)
	start := strings.Index(content, "description: |\n")
	require.Greater(t, start, 0)

	descStart := start + len("description: |\n")
	rest := content[descStart:]

	var descLines []string
	for _, line := range strings.Split(rest, "\n") {
		if strings.HasPrefix(line, "  ") {
			descLines = append(descLines, strings.TrimPrefix(line, "  "))
		} else {
			break
		}
	}
	desc := strings.Join(descLines, "\n")
	assert.Less(t, len(desc), 1024, "description should be under 1024 chars")
	assert.NotEmpty(t, desc)
}

func TestSkill_FlagsTablePopulated(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "| Flag | Type | Default | Required | Description |")
	assert.Contains(t, content, "`--port`")
	assert.Contains(t, content, "`--host`")
}

func TestSkill_RequiredFlagMarked(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	content := string(out)
	serveIdx := strings.Index(content, "#### `myapp serve`")
	require.Greater(t, serveIdx, 0)
	serveSection := content[serveIdx:]

	portLine := findTableLine(serveSection, "--port")
	require.NotEmpty(t, portLine, "should find port in flags table")
	assert.Contains(t, portLine, "| yes |")
}

func TestSkill_EnvVarsTablePopulated(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	content := string(out)
	// structcli.Define() creates env vars from the command path (no app prefix without SetupConfig)
	assert.Contains(t, content, "**Environment Variables:**")
	assert.Contains(t, content, "| Variable | Flag | Description |")
	assert.Contains(t, content, "SERVE_PORT")
	assert.Contains(t, content, "SERVE_HOST")
}

func TestSkill_ExamplesSectionPresent(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "### Examples")
	assert.Contains(t, content, "myapp serve --port 8080 --host 0.0.0.0")
}

func TestSkill_ExamplesSectionOmittedWhenEmpty(t *testing.T) {
	noop := func(cmd *cobra.Command, args []string) error { return nil }
	root := &cobra.Command{Use: "noexample", Short: "No examples", RunE: noop}
	sub := &cobra.Command{Use: "sub", Short: "Sub command", RunE: noop}
	sub.Flags().String("flag1", "", "A flag")
	root.AddCommand(sub)

	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	assert.NotContains(t, string(out), "### Examples")
}

func TestSkill_MetadataFields(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Skill(root, generate.SkillOptions{
		Author:    "test-author",
		Version:   "1.0.0",
		MCPServer: "my-server",
	})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "metadata:")
	assert.Contains(t, content, "author: test-author")
	assert.Contains(t, content, "version: 1.0.0")
	assert.Contains(t, content, "mcp-server: my-server")
}

func TestSkill_MetadataOmittedWhenEmpty(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	assert.NotContains(t, string(out), "metadata:")
}

func TestSkill_NoXMLTags(t *testing.T) {
	noop := func(cmd *cobra.Command, args []string) error { return nil }
	root := &cobra.Command{
		Use:   "xmltest",
		Short: "Test <xml> tags in <description>",
		RunE:  noop,
	}

	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	content := string(out)
	start := strings.Index(content, "description: |\n")
	require.Greater(t, start, 0)
	descStart := start + len("description: |\n")
	rest := content[descStart:]
	var descLines []string
	for _, line := range strings.Split(rest, "\n") {
		if strings.HasPrefix(line, "  ") {
			descLines = append(descLines, strings.TrimPrefix(line, "  "))
		} else {
			break
		}
	}
	desc := strings.Join(descLines, "\n")
	assert.NotContains(t, desc, "<")
	assert.NotContains(t, desc, ">")
}

func TestSkill_CommandPathInBody(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "#### `myapp`")
	assert.Contains(t, content, "#### `myapp serve`")
	assert.Contains(t, content, "#### `myapp config`")
}

func TestSkill_MinimalCLI_NoFlags(t *testing.T) {
	root := buildMinimalTree()
	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "name: bare")
	assert.Contains(t, content, "#### `bare`")
	// No flags table since there are no flags
	assert.NotContains(t, content, "| Flag |")
}

func TestSkill_EmptyDescription(t *testing.T) {
	root := buildNoDescriptionTree()
	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	content := string(out)
	// Should not crash, should produce valid output
	assert.Contains(t, content, "name: nodesc")
}

func TestSkill_YAMLSpecialCharsInMetadata(t *testing.T) {
	root := buildMinimalTree()
	out, err := generate.Skill(root, generate.SkillOptions{
		Author:  "Alice: The Author",
		Version: "1.0.0",
	})
	require.NoError(t, err)

	content := string(out)
	// Author with colon should be quoted
	assert.Contains(t, content, `author: "Alice: The Author"`)
	// Version without special chars should NOT be quoted
	assert.Contains(t, content, "version: 1.0.0")
}

func TestSkill_EmptyFlagDescription(t *testing.T) {
	noop := func(cmd *cobra.Command, args []string) error { return nil }
	root := &cobra.Command{Use: "app", Short: "App", RunE: noop}
	root.Flags().String("silent", "", "")

	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	content := string(out)
	// Empty flag description should show "-" not empty
	line := findTableLine(content, "silent")
	assert.Contains(t, line, "| - |")
}

func TestSkill_CommandsSorted(t *testing.T) {
	root := buildTestTree()
	out, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)

	content := string(out)
	configIdx := strings.Index(content, "#### `myapp config`")
	serveIdx := strings.Index(content, "#### `myapp serve`")
	assert.Greater(t, configIdx, 0)
	assert.Greater(t, serveIdx, 0)
	// config comes before serve alphabetically
	assert.Less(t, configIdx, serveIdx)
}

// --- helpers ---

func findTableLine(section, substr string) string {
	for _, line := range strings.Split(section, "\n") {
		if strings.Contains(line, substr) && strings.HasPrefix(line, "|") {
			return line
		}
	}
	return ""
}
