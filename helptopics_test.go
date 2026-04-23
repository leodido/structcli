package structcli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/config"
	"github.com/leodido/structcli/helptopics"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Option structs for help topic tests ---

type helpTopicGlobalOptions struct {
	Verbose bool   `flag:"verbose" flagdescr:"Enable verbose output" default:"false" flagenv:"true"`
	Output  string `flag:"output" flagdescr:"Output format" default:"text"`
}

func (o *helpTopicGlobalOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

type helpTopicServeOptions struct {
	Port    int    `flag:"port" flagdescr:"Listen port" default:"8080" flagenv:"true"`
	Host    string `flag:"host" flagdescr:"Bind address" default:"localhost" flagenv:"true"`
	TLSCert string `flag:"tls-cert" flagdescr:"TLS certificate path"`
}

func (o *helpTopicServeOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

type helpTopicEnvOnlyOptions struct {
	Secret string `flagenv:"only" flag:"secret" flagdescr:"A secret value"`
}

func (o *helpTopicEnvOnlyOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

type helpTopicEmbeddedOptions struct {
	Auth helpTopicAuth `flag:"auth"`
}

type helpTopicAuth struct {
	User string `flag:"user" flagdescr:"Username" flagenv:"true"`
	Pass string `flag:"pass" flagdescr:"Password" flagenv:"true"`
}

func (o *helpTopicEmbeddedOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

// --- Test helpers ---

// noop is a minimal RunE so cobra treats the command as executable (not a help topic).
var noop = func(cmd *cobra.Command, args []string) error { return nil }

func setupHelpTopicCmd(t *testing.T) *cobra.Command {
	t.Helper()

	structcli.SetEnvPrefix("MYAPP")

	root := &cobra.Command{Use: "myapp", Short: "Test app"}
	globalOpts := &helpTopicGlobalOptions{}
	require.NoError(t, globalOpts.Attach(root))

	serve := &cobra.Command{Use: "serve", Short: "Start server", RunE: noop}
	serveOpts := &helpTopicServeOptions{}
	require.NoError(t, serveOpts.Attach(serve))
	root.AddCommand(serve)

	return root
}

// runHelpTopicCmd executes a help topic subcommand and returns its stdout.
func runHelpTopicCmd(t *testing.T, root *cobra.Command, args ...string) string {
	t.Helper()

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	require.NoError(t, root.Execute())

	return buf.String()
}

// --- SetupHelpTopics tests ---

func TestSetupHelpTopics_AddsCommands(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	// Verify both commands exist and are annotated as help topics.
	envCmd, _, err := root.Find([]string{"env-vars"})
	require.NoError(t, err)
	assert.Equal(t, "env-vars", envCmd.Use)
	assert.True(t, structcli.IsHelpTopicCommand(envCmd), "env-vars should be annotated as help topic")

	cfgCmd, _, err := root.Find([]string{"config-keys"})
	require.NoError(t, err)
	assert.Equal(t, "config-keys", cfgCmd.Use)
	assert.True(t, structcli.IsHelpTopicCommand(cfgCmd), "config-keys should be annotated as help topic")
}

func TestSetupHelpTopics_EnvVars_ContainsGlobalFlags(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "env-vars")

	assert.Contains(t, out, "Environment Variables")
	assert.Contains(t, out, "myapp (global)")
	assert.Contains(t, out, "--verbose")
	assert.Contains(t, out, "MYAPP_VERBOSE")
}

func TestSetupHelpTopics_EnvVars_ContainsSubcommandFlags(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "env-vars")

	assert.Contains(t, out, "myapp serve")
	assert.Contains(t, out, "--port")
	assert.Contains(t, out, "--host")
	assert.Contains(t, out, "MYAPP_SERVE_PORT")
	assert.Contains(t, out, "MYAPP_SERVE_HOST")
}

func TestSetupHelpTopics_EnvVars_OmitsFlagsWithoutEnv(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "env-vars")

	// --output has no flagenv, should not appear.
	assert.NotContains(t, out, "--output")
	// --tls-cert has no flagenv, should not appear.
	assert.NotContains(t, out, "--tls-cert")
}

func TestSetupHelpTopics_EnvVars_EnvOnlyMarker(t *testing.T) {
	structcli.SetEnvPrefix("MYAPP")

	root := &cobra.Command{Use: "myapp"}
	opts := &helpTopicEnvOnlyOptions{}
	require.NoError(t, opts.Attach(root))

	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "env-vars")

	assert.Contains(t, out, "(env-only)")
	assert.Contains(t, out, "MYAPP_SECRET")
}

func TestSetupHelpTopics_EnvVars_ShowsTypeAndDefault(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "env-vars")

	// Port should show int type and default 8080.
	assert.Contains(t, out, "int")
	assert.Contains(t, out, "8080")
}

func TestSetupHelpTopics_ConfigKeys_ContainsGlobalFlags(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "config-keys")

	assert.Contains(t, out, "Configuration Keys")
	assert.Contains(t, out, "myapp (global)")
	assert.Contains(t, out, "verbose")
	assert.Contains(t, out, "output")
}

func TestSetupHelpTopics_ConfigKeys_ContainsSubcommandFlags(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "config-keys")

	assert.Contains(t, out, "myapp serve")
	assert.Contains(t, out, "port")
	assert.Contains(t, out, "host")
	assert.Contains(t, out, "tls-cert")
}

func TestSetupHelpTopics_ConfigKeys_ExcludesHiddenFlags(t *testing.T) {
	structcli.SetEnvPrefix("MYAPP")

	root := &cobra.Command{Use: "myapp"}
	opts := &helpTopicEnvOnlyOptions{}
	require.NoError(t, opts.Attach(root))

	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "config-keys")

	// env-only flags are hidden, so they should not appear in config-keys.
	assert.NotContains(t, out, "secret")
}

func TestSetupHelpTopics_ConfigKeys_ShowsConfigFlag(t *testing.T) {
	structcli.SetEnvPrefix("MYAPP")

	root := &cobra.Command{Use: "myapp"}
	globalOpts := &helpTopicGlobalOptions{}
	require.NoError(t, globalOpts.Attach(root))

	structcli.SetupConfig(root, config.Options{
		AppName:  "myapp",
		FlagName: "config",
	})

	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "config-keys")

	assert.Contains(t, out, "Config flag: --config")
}

func TestSetupHelpTopics_ConfigKeys_AliasFromStructPath(t *testing.T) {
	structcli.SetEnvPrefix("MYAPP")

	root := &cobra.Command{Use: "myapp"}
	opts := &helpTopicEmbeddedOptions{}
	require.NoError(t, opts.Attach(root))

	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "config-keys")

	// The struct path "auth.user" differs from flag name "user",
	// so it should appear as an alias.
	assert.Contains(t, out, "alias for")
}

func TestSetupHelpTopics_SkipsHelpTopicCommands(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "env-vars")

	// The help topic commands themselves should not appear as subcommands.
	assert.NotContains(t, out, "env-vars:")
	assert.NotContains(t, out, "config-keys:")
}

func TestSetupHelpTopics_SkipsHiddenCommands(t *testing.T) {
	structcli.SetEnvPrefix("MYAPP")

	root := &cobra.Command{Use: "myapp"}
	globalOpts := &helpTopicGlobalOptions{}
	require.NoError(t, globalOpts.Attach(root))

	hidden := &cobra.Command{Use: "internal", Hidden: true, RunE: noop}
	hiddenOpts := &helpTopicServeOptions{}
	require.NoError(t, hiddenOpts.Attach(hidden))
	root.AddCommand(hidden)

	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "env-vars")

	assert.NotContains(t, out, "myapp internal")
}

func TestSetupHelpTopics_NoEnvBindings_NoSection(t *testing.T) {
	root := &cobra.Command{Use: "myapp"}
	// No flags defined at all.
	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "env-vars")

	// Should have the header but no command sections.
	assert.Contains(t, out, "Environment Variables")
	assert.NotContains(t, out, "myapp (global)")
}

func TestSetupHelpTopics_AccessibleViaHelp(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	// "myapp help env-vars" shows the Long description (static summary).
	out := runHelpTopicCmd(t, root, "help", "env-vars")

	assert.Contains(t, out, "environment-variable mapping")
}

func TestSetupHelpTopics_NestedSubcommands(t *testing.T) {
	structcli.SetEnvPrefix("MYAPP")

	root := &cobra.Command{Use: "myapp"}
	globalOpts := &helpTopicGlobalOptions{}
	require.NoError(t, globalOpts.Attach(root))

	parent := &cobra.Command{Use: "cluster", Short: "Cluster management"}
	root.AddCommand(parent)

	child := &cobra.Command{Use: "create", Short: "Create cluster", RunE: noop}
	childOpts := &helpTopicServeOptions{}
	require.NoError(t, childOpts.Attach(child))
	parent.AddCommand(child)

	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "env-vars")

	assert.Contains(t, out, "myapp cluster create")
	// Env var names use the leaf command name, not the full path.
	assert.Contains(t, out, "MYAPP_CREATE_PORT")
}

func TestSetupHelpTopics_ConfigKeys_NestingHint(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "config-keys")

	assert.Contains(t, out, "Keys can be nested under the command name in the config file.")
}

func TestSetupHelpTopics_RejectsNonRoot(t *testing.T) {
	root := &cobra.Command{Use: "myapp"}
	child := &cobra.Command{Use: "sub", RunE: noop}
	root.AddCommand(child)

	err := structcli.SetupHelpTopics(child, helptopics.Options{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "root command")
}

func TestSetupHelpTopics_EnvVars_AliasEnvVarShowsMapping(t *testing.T) {
	structcli.SetEnvPrefix("MYAPP")

	root := &cobra.Command{Use: "myapp"}
	globalOpts := &helpTopicGlobalOptions{}
	require.NoError(t, globalOpts.Attach(root))

	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "env-vars")

	// Every env var line should have either "--" (primary) or "(alias for" (secondary).
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "Environment") ||
			strings.HasSuffix(trimmed, ":") {
			continue
		}
		assert.True(t, strings.Contains(trimmed, "--") || strings.Contains(trimmed, "(alias for"),
			"orphan env var line with no context: %q", trimmed)
	}
}

func TestSetupHelpTopics_LazyGeneration(t *testing.T) {
	structcli.SetEnvPrefix("MYAPP")

	root := &cobra.Command{Use: "myapp"}
	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	// Add a subcommand AFTER SetupHelpTopics.
	late := &cobra.Command{Use: "late", Short: "Added late", RunE: noop}
	lateOpts := &helpTopicServeOptions{}
	require.NoError(t, lateOpts.Attach(late))
	root.AddCommand(late)

	out := runHelpTopicCmd(t, root, "env-vars")

	// The late command should appear because generation is lazy.
	assert.Contains(t, out, "myapp late")
	assert.Contains(t, out, "MYAPP_LATE_PORT")
}

func TestSetupHelpTopics_ReferenceSection(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{ReferenceSection: true}))

	// Check that --help shows Reference: section.
	out := runHelpTopicCmd(t, root, "--help")

	assert.Contains(t, out, "Reference:")
	assert.Contains(t, out, "env-vars")
	assert.Contains(t, out, "config-keys")
	// They should NOT appear under Available Commands.
	lines := strings.Split(out, "\n")
	inAvailable := false
	for _, line := range lines {
		if strings.HasPrefix(line, "Available Commands:") {
			inAvailable = true

			continue
		}
		if strings.HasPrefix(line, "Reference:") || strings.HasPrefix(line, "Flags:") ||
			strings.HasPrefix(line, "Global Flags:") || line == "" {
			inAvailable = false

			continue
		}
		if inAvailable {
			assert.NotContains(t, line, "env-vars", "env-vars should not be under Available Commands")
			assert.NotContains(t, line, "config-keys", "config-keys should not be under Available Commands")
		}
	}
}

func TestSetupHelpTopics_DefaultShowsInAvailableCommands(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root, helptopics.Options{}))

	out := runHelpTopicCmd(t, root, "--help")

	// Default: no Reference section, help topics appear as regular commands.
	assert.NotContains(t, out, "Reference:")
	// They should appear under Available Commands.
	lines := strings.Split(out, "\n")
	inAvailable := false
	foundEnvVars, foundConfigKeys := false, false
	for _, line := range lines {
		if strings.HasPrefix(line, "Available Commands:") {
			inAvailable = true

			continue
		}
		if inAvailable && (strings.HasPrefix(line, "Flags:") ||
			strings.HasPrefix(line, "Global Flags:") || line == "") {
			inAvailable = false

			continue
		}
		if inAvailable {
			if strings.Contains(line, "env-vars") {
				foundEnvVars = true
			}
			if strings.Contains(line, "config-keys") {
				foundConfigKeys = true
			}
		}
	}
	assert.True(t, foundEnvVars, "env-vars should appear under Available Commands by default")
	assert.True(t, foundConfigKeys, "config-keys should appear under Available Commands by default")
}
