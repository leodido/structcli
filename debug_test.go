package structcli_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/debug"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type debugTestOptions struct {
	Port    int    `flag:"port" flagdescr:"Listen port" default:"8080" flagenv:"true"`
	Verbose bool   `flag:"verbose" flagdescr:"Verbose output" default:"false"`
	Level   string `flag:"log-level" flagdescr:"Log level" default:"info" flagenv:"true"`
}

func (o *debugTestOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

func setupDebugCmd(t *testing.T, args []string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()

	var buf bytes.Buffer
	opts := &debugTestOptions{}

	root := &cobra.Command{Use: "testapp"}
	cmd := &cobra.Command{
		Use: "serve",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Unmarshal triggers UseDebug automatically.
			return structcli.Unmarshal(cmd, opts)
		},
	}
	root.AddCommand(cmd)

	structcli.SetupDebug(root, debug.Options{AppName: "testapp"})
	require.NoError(t, opts.Attach(cmd))

	root.SetOut(&buf)
	root.SetArgs(args)

	return root, &buf
}

func TestUseDebug_TextOutput(t *testing.T) {
	root, buf := setupDebugCmd(t, []string{"serve", "--debug-options", "--port", "9090"})

	require.NoError(t, root.Execute())

	output := buf.String()
	assert.Contains(t, output, "Command: testapp serve")
	assert.Contains(t, output, "Flags:")
	assert.Contains(t, output, "--port")
	assert.Contains(t, output, "9090")
	assert.Contains(t, output, "(flag)")
	assert.Contains(t, output, "Values:")
}

func TestUseDebug_TextOutput_DefaultSource(t *testing.T) {
	root, buf := setupDebugCmd(t, []string{"serve", "--debug-options"})

	require.NoError(t, root.Execute())

	output := buf.String()
	assert.Contains(t, output, "(default)")
}

func TestUseDebug_TextOutput_EnvSource(t *testing.T) {
	t.Setenv("TESTAPP_SERVE_PORT", "7070")

	root, buf := setupDebugCmd(t, []string{"serve", "--debug-options"})

	require.NoError(t, root.Execute())

	output := buf.String()
	assert.Contains(t, output, "(env: TESTAPP_SERVE_PORT)")
}

func TestUseDebug_JSONOutput(t *testing.T) {
	root, buf := setupDebugCmd(t, []string{"serve", "--debug-options=json", "--port", "9090"})
	require.NoError(t, root.Execute())

	// Verify valid JSON.
	var result map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	// Verify structure.
	assert.Equal(t, "testapp serve", result["command"])
	assert.Contains(t, result, "flags")
	assert.Contains(t, result, "values")

	// Verify flags array.
	flags, ok := result["flags"].([]any)
	require.True(t, ok)

	// Find the port flag.
	var portFlag map[string]any
	for _, f := range flags {
		fm := f.(map[string]any)
		if fm["name"] == "port" {
			portFlag = fm
			break
		}
	}
	require.NotNil(t, portFlag, "port flag should be in output")
	assert.Equal(t, "9090", portFlag["value"])
	assert.Equal(t, "8080", portFlag["default"])
	assert.Equal(t, true, portFlag["changed"])
	assert.Equal(t, "flag", portFlag["source"])
}

func TestUseDebug_JSONOutput_SourceAttribution(t *testing.T) {
	t.Setenv("TESTAPP_SERVE_LOG_LEVEL", "debug")

	root, buf := setupDebugCmd(t, []string{"serve", "--debug-options=json", "--port", "9090"})
	require.NoError(t, root.Execute())

	var result map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	flags := result["flags"].([]any)

	sources := map[string]string{}
	for _, f := range flags {
		fm := f.(map[string]any)
		sources[fm["name"].(string)] = fm["source"].(string)
	}

	assert.Equal(t, "flag", sources["port"], "port was set via --port")
	assert.Equal(t, "env", sources["log-level"], "log-level was set via TESTAPP_SERVE_LOG_LEVEL")
	assert.Equal(t, "default", sources["verbose"], "verbose was not set")
}

func TestUseDebug_EnvVarActivation_JSON(t *testing.T) {
	t.Setenv("TESTAPP_DEBUG_OPTIONS", "json")

	root, buf := setupDebugCmd(t, []string{"serve"})
	require.NoError(t, root.Execute())

	var result map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "testapp serve", result["command"])
}

func TestUseDebug_EnvVarActivation_TruthyText(t *testing.T) {
	t.Setenv("TESTAPP_DEBUG_OPTIONS", "1")

	root, buf := setupDebugCmd(t, []string{"serve"})
	require.NoError(t, root.Execute())

	output := buf.String()
	assert.Contains(t, output, "Command: testapp serve")
	assert.Contains(t, output, "Flags:")
}

func TestUseDebug_Inactive(t *testing.T) {
	root, buf := setupDebugCmd(t, []string{"serve"})
	require.NoError(t, root.Execute())

	assert.Empty(t, buf.String(), "no debug output when flag not set")
}

func TestIsDebugActive(t *testing.T) {
	root, _ := setupDebugCmd(t, []string{"serve", "--debug-options"})
	require.NoError(t, root.Execute())

	// The public wrapper should reflect the internal state.
	cmd, _, _ := root.Find([]string{"serve"})
	assert.True(t, structcli.IsDebugActive(cmd))
}

func TestIsDebugActive_False(t *testing.T) {
	root, _ := setupDebugCmd(t, []string{"serve"})
	require.NoError(t, root.Execute())

	cmd, _, _ := root.Find([]string{"serve"})
	assert.False(t, structcli.IsDebugActive(cmd))
}

func TestSetupDebug_WorksWithoutRunE(t *testing.T) {
	// Verify that --debug-options on a root command without RunE/Run doesn't
	// short-circuit to Help(). EnsureRunnable sets a synthetic RunE so cobra
	// calls PreRunE, and RecursivelyWrapRun intercepts execution when debug
	// is active (returning nil instead of running the original).
	root := &cobra.Command{
		Use:           "app",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	require.NoError(t, structcli.SetupDebug(root, debug.Options{
		AppName: "app",
		Exit:    true,
	}))

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--debug-options"})
	require.NoError(t, root.Execute())

	output := out.String()
	// The wrapped RunE returns nil when debug is active, so the synthetic
	// Help() body never runs — output should be empty (no help text).
	assert.NotContains(t, output, "Usage:", "debug interception should prevent help output")
}

func TestUseDebug_HiddenFlagsExcluded(t *testing.T) {
	var buf bytes.Buffer
	opts := &debugTestOptions{}

	root := &cobra.Command{Use: "testapp"}
	cmd := &cobra.Command{
		Use: "serve",
		RunE: func(cmd *cobra.Command, args []string) error {
			return structcli.Unmarshal(cmd, opts)
		},
	}
	root.AddCommand(cmd)

	structcli.SetupDebug(root, debug.Options{AppName: "testapp"})
	require.NoError(t, opts.Attach(cmd))

	// Add a hidden flag after Define.
	cmd.Flags().String("internal-token", "", "internal use only")
	cmd.Flags().MarkHidden("internal-token")

	root.SetOut(&buf)
	root.SetArgs([]string{"serve", "--debug-options", "--internal-token", "secret"})
	require.NoError(t, root.Execute())

	output := buf.String()
	assert.NotContains(t, output, "internal-token", "hidden flags should not appear in debug output")
	assert.NotContains(t, output, "secret", "hidden flag values should not appear in debug output")
	// Non-hidden flags should still be present.
	assert.Contains(t, output, "--port")
}

func TestUseDebug_JSONOutput_HiddenFlagsExcluded(t *testing.T) {
	// Verify hidden flags are also excluded from JSON output.
	var buf bytes.Buffer
	opts := &debugTestOptions{}

	root := &cobra.Command{Use: "testapp"}
	cmd := &cobra.Command{
		Use: "serve",
		RunE: func(cmd *cobra.Command, args []string) error {
			return structcli.Unmarshal(cmd, opts)
		},
	}
	root.AddCommand(cmd)

	structcli.SetupDebug(root, debug.Options{AppName: "testapp"})
	require.NoError(t, opts.Attach(cmd))

	cmd.Flags().String("internal-token", "", "internal use only")
	require.NoError(t, cmd.Flags().MarkHidden("internal-token"))

	root.SetOut(&buf)
	root.SetArgs([]string{"serve", "--debug-options=json", "--internal-token", "secret"})
	require.NoError(t, root.Execute())

	var result map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	flags := result["flags"].([]any)
	for _, f := range flags {
		fm := f.(map[string]any)
		assert.NotEqual(t, "internal-token", fm["name"], "hidden flags should not appear in JSON output")
	}
}
