package generate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/generate"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testServeOptions defines flags via structcli.Define(), the same path real users take.
// This ensures annotations (env vars, defaults, required) are set correctly.
type testServeOptions struct {
	Port int    `flagshort:"p" flagdescr:"Server port" flagenv:"true" flagrequired:"true" default:"3000"`
	Host string `flag:"host" flagdescr:"Server host" flagenv:"true" default:"localhost"`
}

func (o *testServeOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

// testConfigOptions for a secondary subcommand.
type testConfigOptions struct {
	Format string `flag:"format" flagdescr:"Output format" default:"json"`
}

func (o *testConfigOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

// buildTestTree creates a realistic CLI tree using structcli.Define().
// All annotations (env vars, defaults, required, paths) are set automatically.
func buildTestTree() *cobra.Command {
	noop := func(cmd *cobra.Command, args []string) error { return nil }

	root := &cobra.Command{
		Use:   "myapp",
		Short: "A test CLI application",
		RunE:  noop,
	}

	serve := &cobra.Command{
		Use:     "serve",
		Short:   "Start the server",
		Example: "myapp serve --port 8080 --host 0.0.0.0",
		RunE:    noop,
	}
	serveOpts := &testServeOptions{}
	serveOpts.Attach(serve)

	config := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		RunE:  noop,
	}
	configOpts := &testConfigOptions{}
	configOpts.Attach(config)

	root.AddCommand(serve, config)

	return root
}

// buildMinimalTree creates a CLI with a single command, no flags, no subcommands.
func buildMinimalTree() *cobra.Command {
	return &cobra.Command{
		Use:   "bare",
		Short: "A bare CLI with no subcommands",
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}

// buildNoDescriptionTree creates a CLI where some commands have empty descriptions.
func buildNoDescriptionTree() *cobra.Command {
	root := &cobra.Command{
		Use:  "nodesc",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}

	sub := &cobra.Command{
		Use:  "sub",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}

	opts := &testServeOptions{}
	opts.Attach(sub)
	root.AddCommand(sub)

	return root
}

// buildDuplicateNameTree creates a CLI where two subcommands share the same name
// under different parents (eg. "db add" and "user add").
func buildDuplicateNameTree() *cobra.Command {
	noop := func(cmd *cobra.Command, args []string) error { return nil }

	root := &cobra.Command{Use: "mycli", Short: "CLI with duplicate names", RunE: noop}

	db := &cobra.Command{Use: "db", Short: "Database commands"}
	dbAdd := &cobra.Command{Use: "add", Short: "Add a database", RunE: noop}
	db.AddCommand(dbAdd)

	user := &cobra.Command{Use: "user", Short: "User commands"}
	userAdd := &cobra.Command{Use: "add", Short: "Add a user", RunE: noop}
	user.AddCommand(userAdd)

	root.AddCommand(db, user)

	return root
}

// buildRunnableParentTree creates a CLI where a parent command is directly
// runnable via Run while also having subcommands. This matches valid Cobra
// behavior and guards against leaf-only discovery heuristics.
func buildRunnableParentTree() *cobra.Command {
	noop := func(cmd *cobra.Command, args []string) error { return nil }

	root := &cobra.Command{Use: "myapp", Short: "A CLI with runnable parents"}

	srv := &cobra.Command{
		Use:   "srv",
		Short: "Start the server",
		Run: func(cmd *cobra.Command, args []string) {
			_ = args
		},
	}
	srv.Flags().Int("port", 3000, "Server port")

	version := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE:  noop,
	}
	srv.AddCommand(version)
	root.AddCommand(srv)

	return root
}

// TestWriteAll_CreatesAllThreeFiles verifies that WriteAll produces SKILL.md,
// llms.txt, and AGENTS.md in the given output directory.
func TestWriteAll_CreatesAllThreeFiles(t *testing.T) {
	root := buildTestTree()
	outDir := t.TempDir()

	err := generate.WriteAll(root, outDir, generate.AllOptions{
		ModulePath: "github.com/example/myapp",
		Skill: generate.SkillOptions{
			Author:  "testauthor",
			Version: "1.2.3",
		},
	})
	require.NoError(t, err)

	for _, name := range []string{"SKILL.md", "llms.txt", "AGENTS.md"} {
		data, err := os.ReadFile(filepath.Join(outDir, name))
		require.NoError(t, err, "reading %s", name)
		assert.NotEmpty(t, data, "%s should not be empty", name)
	}
}

// TestWriteAll_SkillFrontmatter verifies that SKILL.md contains the expected
// frontmatter fields when options are passed through AllOptions.
func TestWriteAll_SkillFrontmatter(t *testing.T) {
	root := buildTestTree()
	outDir := t.TempDir()

	require.NoError(t, generate.WriteAll(root, outDir, generate.AllOptions{
		ModulePath: "github.com/example/myapp",
		Skill: generate.SkillOptions{
			Author:  "leodido",
			Version: "0.9.0",
		},
	}))

	data, err := os.ReadFile(filepath.Join(outDir, "SKILL.md"))
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "name: myapp")
	assert.Contains(t, content, "author: leodido")
	assert.Contains(t, content, "version: 0.9.0")
}

// TestWriteAll_ModulePathPropagated verifies that ModulePath from AllOptions
// is propagated into llms.txt and AGENTS.md.
func TestWriteAll_ModulePathPropagated(t *testing.T) {
	root := buildTestTree()
	outDir := t.TempDir()
	modulePath := "github.com/myorg/myprojcli"

	require.NoError(t, generate.WriteAll(root, outDir, generate.AllOptions{
		ModulePath: modulePath,
	}))

	for _, name := range []string{"llms.txt", "AGENTS.md"} {
		data, err := os.ReadFile(filepath.Join(outDir, name))
		require.NoError(t, err)
		assert.Contains(t, string(data), modulePath, "%s should reference ModulePath", name)
	}
}

// TestWriteAll_ErrorOnBadDir verifies that WriteAll returns an error when
// the output directory does not exist.
func TestWriteAll_ErrorOnBadDir(t *testing.T) {
	root := buildMinimalTree()
	err := generate.WriteAll(root, "/nonexistent/path/that/does/not/exist/at/all", generate.AllOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "writing SKILL.md")
}

// --- Env-only field tests ---

type testEnvOnlyOptions struct {
	APIKey string `flagenv:"only" flag:"api-key" flagdescr:"API secret key" flagrequired:"true"`
	Port   int    `flagshort:"p" flagdescr:"Server port" flagenv:"true" default:"3000"`
}

func (o *testEnvOnlyOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

func buildEnvOnlyTree() *cobra.Command {
	structcli.SetEnvPrefix("APP")
	noop := func(cmd *cobra.Command, args []string) error { return nil }

	root := &cobra.Command{
		Use:   "envapp",
		Short: "App with env-only fields",
		RunE:  noop,
	}

	serve := &cobra.Command{
		Use:   "serve",
		Short: "Start the server",
		RunE:  noop,
	}
	opts := &testEnvOnlyOptions{}
	opts.Attach(serve)
	root.AddCommand(serve)

	return root
}

func TestAgents_EnvOnlyExcludedFromFlagTable(t *testing.T) {
	root := buildEnvOnlyTree()
	data, err := generate.Agents(root, generate.AgentsOptions{})
	require.NoError(t, err)
	output := string(data)

	// env-only field should NOT appear in the flag table
	assert.NotContains(t, output, "| `--api-key`", "env-only field should not appear in flag table")
	// normal flag should appear
	assert.Contains(t, output, "| `--port`", "normal flag should appear in flag table")
	// env-only field should appear in env var table with "(env only)" marker
	assert.Contains(t, output, "*(env only)*", "env-only field should have env-only marker in env table")
	assert.Contains(t, output, "APP_SERVE_API_KEY", "env-only field's env var should appear")
}

func TestSkill_EnvOnlyExcludedFromFlagTable(t *testing.T) {
	root := buildEnvOnlyTree()
	data, err := generate.Skill(root, generate.SkillOptions{})
	require.NoError(t, err)
	output := string(data)

	// env-only field should NOT appear in the flags table
	assert.NotContains(t, output, "| `--api-key`", "env-only field should not appear in flag table")
	// normal flag should appear
	assert.Contains(t, output, "| `--port`", "normal flag should appear in flag table")
	// env-only field should appear in env var table
	assert.Contains(t, output, "*(env only)*", "env-only field should have env-only marker")
	assert.Contains(t, output, "APP_SERVE_API_KEY", "env-only field's env var should appear")
}

func TestLLMSTxt_EnvOnlyExcludedFromFlagSection(t *testing.T) {
	root := buildEnvOnlyTree()
	data, err := generate.LLMsTxt(root, generate.LLMsTxtOptions{})
	require.NoError(t, err)
	output := string(data)

	// env-only field should NOT appear in flags section
	assert.NotContains(t, output, "- `--api-key`", "env-only field should not appear in flags section")
	// normal flag should appear
	assert.Contains(t, output, "- `--port`", "normal flag should appear in flags section")
	// env-only field should appear in env vars section
	assert.Contains(t, output, "env only", "env-only field should have env-only marker")
	assert.Contains(t, output, "APP_SERVE_API_KEY", "env-only field's env var should appear")
}
