package structcli

import (
	"testing"

	"github.com/leodido/structcli/debug"
	"github.com/leodido/structcli/helptopics"
	"github.com/leodido/structcli/jsonschema"
	structclimcp "github.com/leodido/structcli/mcp"
	"github.com/spf13/cobra"
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
	// SetupFlagErrors doesn't register flags — it intercepts errors.
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
