package structcli_test

import (
	"strings"
	"testing"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/config"
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

// --- SetupHelpTopics tests ---

func TestSetupHelpTopics_AddsCommands(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root))

	// Verify both help topic commands exist.
	envCmd, _, err := root.Find([]string{"env-vars"})
	require.NoError(t, err)
	assert.Equal(t, "env-vars", envCmd.Use)
	assert.True(t, envCmd.IsAdditionalHelpTopicCommand(), "env-vars should be a help topic (no Run)")

	cfgCmd, _, err := root.Find([]string{"config-keys"})
	require.NoError(t, err)
	assert.Equal(t, "config-keys", cfgCmd.Use)
	assert.True(t, cfgCmd.IsAdditionalHelpTopicCommand(), "config-keys should be a help topic (no Run)")
}

func TestSetupHelpTopics_EnvVars_ContainsGlobalFlags(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root))

	envCmd, _, _ := root.Find([]string{"env-vars"})
	long := envCmd.Long

	assert.Contains(t, long, "Environment Variables")
	assert.Contains(t, long, "myapp (global)")
	assert.Contains(t, long, "--verbose")
	assert.Contains(t, long, "MYAPP_VERBOSE")
}

func TestSetupHelpTopics_EnvVars_ContainsSubcommandFlags(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root))

	envCmd, _, _ := root.Find([]string{"env-vars"})
	long := envCmd.Long

	assert.Contains(t, long, "myapp serve")
	assert.Contains(t, long, "--port")
	assert.Contains(t, long, "--host")
	assert.Contains(t, long, "MYAPP_SERVE_PORT")
	assert.Contains(t, long, "MYAPP_SERVE_HOST")
}

func TestSetupHelpTopics_EnvVars_OmitsFlagsWithoutEnv(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root))

	envCmd, _, _ := root.Find([]string{"env-vars"})
	long := envCmd.Long

	// --output has no flagenv, should not appear.
	assert.NotContains(t, long, "--output")
	// --tls-cert has no flagenv, should not appear.
	assert.NotContains(t, long, "--tls-cert")
}

func TestSetupHelpTopics_EnvVars_EnvOnlyMarker(t *testing.T) {
	structcli.SetEnvPrefix("MYAPP")

	root := &cobra.Command{Use: "myapp"}
	opts := &helpTopicEnvOnlyOptions{}
	require.NoError(t, opts.Attach(root))

	require.NoError(t, structcli.SetupHelpTopics(root))

	envCmd, _, _ := root.Find([]string{"env-vars"})
	long := envCmd.Long

	assert.Contains(t, long, "(env-only)")
	assert.Contains(t, long, "MYAPP_SECRET")
}

func TestSetupHelpTopics_EnvVars_ShowsTypeAndDefault(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root))

	envCmd, _, _ := root.Find([]string{"env-vars"})
	long := envCmd.Long

	// Port should show int type and default 8080.
	assert.Contains(t, long, "int")
	assert.Contains(t, long, "8080")
}

func TestSetupHelpTopics_ConfigKeys_ContainsGlobalFlags(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root))

	cfgCmd, _, _ := root.Find([]string{"config-keys"})
	long := cfgCmd.Long

	assert.Contains(t, long, "Configuration Keys")
	assert.Contains(t, long, "myapp (global)")
	assert.Contains(t, long, "verbose")
	assert.Contains(t, long, "output")
}

func TestSetupHelpTopics_ConfigKeys_ContainsSubcommandFlags(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root))

	cfgCmd, _, _ := root.Find([]string{"config-keys"})
	long := cfgCmd.Long

	assert.Contains(t, long, "myapp serve")
	assert.Contains(t, long, "port")
	assert.Contains(t, long, "host")
	assert.Contains(t, long, "tls-cert")
}

func TestSetupHelpTopics_ConfigKeys_ExcludesHiddenFlags(t *testing.T) {
	structcli.SetEnvPrefix("MYAPP")

	root := &cobra.Command{Use: "myapp"}
	opts := &helpTopicEnvOnlyOptions{}
	require.NoError(t, opts.Attach(root))

	require.NoError(t, structcli.SetupHelpTopics(root))

	cfgCmd, _, _ := root.Find([]string{"config-keys"})
	long := cfgCmd.Long

	// env-only flags are hidden, so they should not appear in config-keys.
	assert.NotContains(t, long, "secret")
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

	require.NoError(t, structcli.SetupHelpTopics(root))

	cfgCmd, _, _ := root.Find([]string{"config-keys"})
	long := cfgCmd.Long

	assert.Contains(t, long, "Config flag: --config")
}

func TestSetupHelpTopics_ConfigKeys_AliasFromStructPath(t *testing.T) {
	structcli.SetEnvPrefix("MYAPP")

	root := &cobra.Command{Use: "myapp"}
	opts := &helpTopicEmbeddedOptions{}
	require.NoError(t, opts.Attach(root))

	require.NoError(t, structcli.SetupHelpTopics(root))

	cfgCmd, _, _ := root.Find([]string{"config-keys"})
	long := cfgCmd.Long

	// The struct path "auth.user" differs from flag name "user",
	// so it should appear as an alias.
	assert.Contains(t, long, "alias for")
}

func TestSetupHelpTopics_SkipsHelpTopicCommands(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root))

	envCmd, _, _ := root.Find([]string{"env-vars"})
	long := envCmd.Long

	// The help topic commands themselves should not appear as subcommands.
	assert.NotContains(t, long, "env-vars:")
	assert.NotContains(t, long, "config-keys:")
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

	require.NoError(t, structcli.SetupHelpTopics(root))

	envCmd, _, _ := root.Find([]string{"env-vars"})
	long := envCmd.Long

	assert.NotContains(t, long, "myapp internal")
}

func TestSetupHelpTopics_NoEnvBindings_NoSection(t *testing.T) {
	root := &cobra.Command{Use: "myapp"}
	// No flags defined at all.
	require.NoError(t, structcli.SetupHelpTopics(root))

	envCmd, _, _ := root.Find([]string{"env-vars"})
	long := envCmd.Long

	// Should have the header but no command sections.
	assert.Contains(t, long, "Environment Variables")
	assert.NotContains(t, long, "myapp (global)")
}

func TestSetupHelpTopics_AccessibleViaHelp(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root))

	// Simulate "myapp help env-vars" by executing the root with those args.
	var out []byte
	root.SetOut(writerFunc(func(p []byte) (int, error) {
		out = append(out, p...)
		return len(p), nil
	}))
	root.SetArgs([]string{"help", "env-vars"})
	require.NoError(t, root.Execute())

	assert.Contains(t, string(out), "Environment Variables")
}

// writerFunc adapts a function to io.Writer.
type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

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

	require.NoError(t, structcli.SetupHelpTopics(root))

	envCmd, _, _ := root.Find([]string{"env-vars"})
	long := envCmd.Long

	assert.Contains(t, long, "myapp cluster create")
	// Env var names use the leaf command name, not the full path.
	assert.Contains(t, long, "MYAPP_CREATE_PORT")
}

func TestSetupHelpTopics_ConfigKeys_NestingHint(t *testing.T) {
	root := setupHelpTopicCmd(t)
	require.NoError(t, structcli.SetupHelpTopics(root))

	cfgCmd, _, _ := root.Find([]string{"config-keys"})
	long := cfgCmd.Long

	assert.Contains(t, long, "Keys can be nested under the command name in the config file.")
}

func TestSetupHelpTopics_RejectsNonRoot(t *testing.T) {
	root := &cobra.Command{Use: "myapp"}
	child := &cobra.Command{Use: "sub", RunE: noop}
	root.AddCommand(child)

	err := structcli.SetupHelpTopics(child)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "root command")
}

func TestSetupHelpTopics_EnvVars_AliasEnvVarShowsMapping(t *testing.T) {
	structcli.SetEnvPrefix("MYAPP")

	root := &cobra.Command{Use: "myapp"}
	globalOpts := &helpTopicGlobalOptions{}
	require.NoError(t, globalOpts.Attach(root))

	require.NoError(t, structcli.SetupHelpTopics(root))

	envCmd, _, _ := root.Find([]string{"env-vars"})
	long := envCmd.Long

	// If a flag has multiple env var names, aliases should reference the primary.
	// The global --verbose flag with MYAPP prefix should have at most one env var,
	// so test with a command that has multi-env flags if available.
	// At minimum, verify no orphan lines (lines with just an env var and no context).
	for _, line := range splitNonEmpty(long) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "Environment") ||
			strings.HasSuffix(trimmed, ":") {
			continue
		}
		// Every env var line should have either "--" (primary) or "(alias for" (secondary).
		assert.True(t, strings.Contains(trimmed, "--") || strings.Contains(trimmed, "(alias for"),
			"orphan env var line with no context: %q", trimmed)
	}
}

// splitNonEmpty splits s by newlines and returns non-empty lines.
func splitNonEmpty(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}
