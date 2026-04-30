package structcli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leodido/structcli/config"
	"github.com/leodido/structcli/debug"
	"github.com/leodido/structcli/helptopics"
	internalenv "github.com/leodido/structcli/internal/env"
	"github.com/leodido/structcli/jsonschema"
	structclimcp "github.com/leodido/structcli/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetup_NoOptions(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd)
	require.NoError(t, err)
}

func TestSetup_WithDebug(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd, WithDebug(debug.Options{}))
	require.NoError(t, err)

	// Debug flag is registered as a persistent flag
	f := cmd.PersistentFlags().Lookup("debug-options")
	assert.NotNil(t, f, "debug-options flag should exist")
}

func TestSetup_WithDebug_CustomFlagName(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd, WithDebug(debug.Options{FlagName: "dbg"}))
	require.NoError(t, err)

	f := cmd.PersistentFlags().Lookup("dbg")
	assert.NotNil(t, f, "custom debug flag name should exist")
}

func TestSetup_WithJSONSchema_Defaults(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd, WithJSONSchema())
	require.NoError(t, err)

	f := cmd.PersistentFlags().Lookup("jsonschema")
	assert.NotNil(t, f, "jsonschema flag should exist")
}

func TestSetup_WithJSONSchema_CustomOptions(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd, WithJSONSchema(jsonschema.Options{FlagName: "schema"}))
	require.NoError(t, err)

	f := cmd.PersistentFlags().Lookup("schema")
	assert.NotNil(t, f, "custom jsonschema flag name should exist")
}

func TestSetup_WithMCP_Defaults(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd, WithMCP())
	require.NoError(t, err)

	f := cmd.PersistentFlags().Lookup("mcp")
	assert.NotNil(t, f, "mcp flag should exist")
}

func TestSetup_WithMCP_CustomOptions(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd, WithMCP(structclimcp.Options{FlagName: "mcp-mode"}))
	require.NoError(t, err)

	f := cmd.PersistentFlags().Lookup("mcp-mode")
	assert.NotNil(t, f, "custom mcp flag name should exist")
}

func TestSetup_WithHelpTopics_Defaults(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd, WithHelpTopics())
	require.NoError(t, err)

	// SetupHelpTopics adds "env-vars" and "config-keys" subcommands
	var names []string
	for _, sub := range cmd.Commands() {
		names = append(names, sub.Name())
	}
	assert.Contains(t, names, "env-vars", "env-vars subcommand should exist")
	assert.Contains(t, names, "config-keys", "config-keys subcommand should exist")
}

func TestSetup_WithFlagErrors(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd, WithFlagErrors())
	require.NoError(t, err)
	// SetupFlagErrors doesn't register flags; it intercepts errors.
}

func TestSetup_AllOptions(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd,
		WithDebug(debug.Options{}),
		WithJSONSchema(),
		WithMCP(),
		WithHelpTopics(helptopics.Options{ReferenceSection: true}),
		WithFlagErrors(),
	)
	require.NoError(t, err)

	assert.NotNil(t, cmd.PersistentFlags().Lookup("debug-options"))
	assert.NotNil(t, cmd.PersistentFlags().Lookup("jsonschema"))
	assert.NotNil(t, cmd.PersistentFlags().Lookup("mcp"))

	var names []string
	for _, sub := range cmd.Commands() {
		names = append(names, sub.Name())
	}
	assert.Contains(t, names, "env-vars")
	assert.Contains(t, names, "config-keys")
}

func TestSetup_CombinesWithBind(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}

	opts := &struct {
		Port int `flag:"port" default:"3000"`
	}{}

	require.NoError(t, Setup(cmd, WithJSONSchema(), WithFlagErrors()))
	require.NoError(t, Bind(cmd, opts))

	cmd.SetArgs([]string{"--port", "8080"})
	_, err := ExecuteC(cmd)
	require.NoError(t, err)

	assert.Equal(t, 8080, opts.Port)
}

// --- WithAppName tests ---

func TestSetup_WithAppName_SetsPrefix(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd, WithAppName("myapp"))
	require.NoError(t, err)

	assert.Equal(t, "MYAPP", EnvPrefix())
}

func TestSetup_WithAppName_PatchesExistingFlags(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}

	opts := &struct {
		Port int `flag:"port" default:"3000" flagenv:"true"`
	}{}

	// Bind first. No global prefix, but command name "test" is baked in.
	// by GetEnv as a pseudo-prefix, so annotation is "TEST_PORT".
	require.NoError(t, Bind(cmd, opts))

	f := cmd.Flags().Lookup("port")
	require.NotNil(t, f)
	envsBefore := f.Annotations[internalenv.FlagAnnotation]
	require.NotEmpty(t, envsBefore)
	assert.Equal(t, "TEST_PORT", envsBefore[0], "before Setup, env uses command name as prefix")

	// Setup with AppName. On root command, the command name pseudo-prefix
	// is replaced by the real app prefix: TEST_PORT → MYAPP_PORT.
	require.NoError(t, Setup(cmd, WithAppName("myapp")))

	envsAfter := f.Annotations[internalenv.FlagAnnotation]
	require.NotEmpty(t, envsAfter)
	assert.Equal(t, "MYAPP_PORT", envsAfter[0], "after Setup, env should use app prefix")
}

func TestSetup_WithAppName_BindBeforeSetup_EnvWorks(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{
		Use: "myapp",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}

	opts := &struct {
		Port int `flag:"port" default:"3000" flagenv:"true"`
	}{}

	// Bind before Setup (ordering independence, AC3).
	require.NoError(t, Bind(cmd, opts))
	require.NoError(t, Setup(cmd, WithAppName("myapp")))

	t.Setenv("MYAPP_PORT", "9090")

	cmd.SetArgs([]string{})
	_, err := ExecuteC(cmd)
	require.NoError(t, err)

	assert.Equal(t, 9090, opts.Port, "env var with app prefix should populate the field")
}

func TestSetup_WithAppName_SetupBeforeBind_EnvWorks(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{
		Use: "myapp",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}

	opts := &struct {
		Port int `flag:"port" default:"3000" flagenv:"true"`
	}{}

	// Setup before Bind (AC4). Prefix is set before Define runs.
	require.NoError(t, Setup(cmd, WithAppName("myapp")))
	require.NoError(t, Bind(cmd, opts))

	t.Setenv("MYAPP_PORT", "7070")

	cmd.SetArgs([]string{})
	_, err := ExecuteC(cmd)
	require.NoError(t, err)

	assert.Equal(t, 7070, opts.Port, "env var with app prefix should populate the field")
}

func TestSetup_WithAppName_PatchesSubcommandFlags(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	root := &cobra.Command{Use: "myapp"}
	child := &cobra.Command{
		Use: "serve",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	root.AddCommand(child)

	opts := &struct {
		Port int `flag:"port" default:"3000" flagenv:"true"`
	}{}

	// Bind on child, then Setup on root.
	require.NoError(t, Bind(child, opts))
	require.NoError(t, Setup(root, WithAppName("myapp")))

	// Child command flags include the command name in the env var.
	f := child.Flags().Lookup("port")
	require.NotNil(t, f)
	envs := f.Annotations[internalenv.FlagAnnotation]
	require.NotEmpty(t, envs)
	assert.Equal(t, "MYAPP_SERVE_PORT", envs[0], "child command env should include command name")
}

func TestSetup_WithAppName_ConflictWithDebug(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd,
		WithAppName("myapp"),
		WithDebug(debug.Options{AppName: "other"}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicts")
}

func TestSetup_WithAppName_ConflictWithConfig(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })
	viper.Reset()
	t.Cleanup(func() { viper.Reset() })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd,
		WithAppName("myapp"),
		WithConfig(config.Options{AppName: "other"}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicts")
}

func TestSetup_WithAppName_SameNameNoConflict(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd,
		WithAppName("myapp"),
		WithDebug(debug.Options{AppName: "myapp"}),
	)
	require.NoError(t, err)
}

func TestSetup_WithAppName_PropagatesIntoSubOptions(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })
	viper.Reset()
	t.Cleanup(func() { viper.Reset() })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd,
		WithAppName("myapp"),
		WithDebug(debug.Options{}),
		WithConfig(config.Options{}),
	)
	require.NoError(t, err)

	// Debug flag should exist (AppName propagated).
	assert.NotNil(t, cmd.PersistentFlags().Lookup("debug-options"))
	// Config flag should exist (AppName propagated).
	assert.NotNil(t, cmd.PersistentFlags().Lookup("config"))
}

// --- WithConfig tests ---

func TestSetup_WithConfig_Defaults(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })
	viper.Reset()
	t.Cleanup(func() { viper.Reset() })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd, WithConfig())
	require.NoError(t, err)

	f := cmd.PersistentFlags().Lookup("config")
	assert.NotNil(t, f, "config flag should exist")
}

func TestSetup_WithConfig_CustomOptions(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })
	viper.Reset()
	t.Cleanup(func() { viper.Reset() })

	cmd := &cobra.Command{Use: "test"}
	err := Setup(cmd, WithConfig(config.Options{FlagName: "settings"}))
	require.NoError(t, err)

	f := cmd.PersistentFlags().Lookup("settings")
	assert.NotNil(t, f, "custom config flag name should exist")
}

func TestSetup_WithConfig_SetsAutoLoadAnnotation(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })
	viper.Reset()
	t.Cleanup(func() { viper.Reset() })

	cmd := &cobra.Command{Use: "test"}
	require.NoError(t, Setup(cmd, WithConfig()))

	assert.Equal(t, "true", cmd.Annotations[configAutoLoadAnnotation],
		"config auto-load annotation should be set on root")
}

func TestSetup_WithConfig_AutoLoadsBeforeUnmarshal(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })
	viper.Reset()
	t.Cleanup(func() { viper.Reset() })

	// Create a temp config file.
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgFile, []byte("port: 4242\n"), 0o644))

	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}

	opts := &struct {
		Port int `flag:"port" default:"3000"`
	}{}

	require.NoError(t, Setup(cmd,
		WithAppName("test"),
		WithConfig(config.Options{}),
	))
	require.NoError(t, Bind(cmd, opts))

	// Point --config at our temp file.
	cmd.SetArgs([]string{"--config", cfgFile})
	_, err := ExecuteC(cmd)
	require.NoError(t, err)

	assert.Equal(t, 4242, opts.Port, "config file value should be loaded before unmarshal")
}

func TestSetup_WithConfig_NoAnnotationWithoutWithConfig(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })

	cmd := &cobra.Command{Use: "test"}
	require.NoError(t, Setup(cmd, WithAppName("myapp")))

	if cmd.Annotations != nil {
		assert.Empty(t, cmd.Annotations[configAutoLoadAnnotation],
			"config auto-load annotation should not be set without WithConfig")
	}
}

// --- Combined WithAppName + WithConfig tests ---

func TestSetup_WithAppNameAndConfig_FullIntegration(t *testing.T) {
	SetEnvPrefix("")
	t.Cleanup(func() { SetEnvPrefix("") })
	viper.Reset()
	t.Cleanup(func() { viper.Reset() })

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgFile, []byte("host: example.com\n"), 0o644))

	root := &cobra.Command{Use: "myapp"}
	child := &cobra.Command{
		Use: "serve",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	root.AddCommand(child)

	opts := &struct {
		Host string `flag:"host" default:"localhost" flagenv:"true"`
	}{}

	require.NoError(t, Bind(child, opts))
	require.NoError(t, Setup(root,
		WithAppName("myapp"),
		WithConfig(config.Options{}),
	))

	root.SetArgs([]string{"serve", "--config", cfgFile})
	_, err := ExecuteC(root)
	require.NoError(t, err)

	assert.Equal(t, "example.com", opts.Host, "config value should be loaded")
	assert.Equal(t, "MYAPP", EnvPrefix(), "prefix should be set")

	// Verify env annotation was patched; child commands include command name.
	f := child.Flags().Lookup("host")
	require.NotNil(t, f)
	envs := f.Annotations[internalenv.FlagAnnotation]
	require.NotEmpty(t, envs)
	assert.Equal(t, "MYAPP_SERVE_HOST", envs[0])
}
