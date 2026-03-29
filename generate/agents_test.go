package generate_test

import (
	"strings"
	"testing"

	"github.com/leodido/structcli/generate"
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
