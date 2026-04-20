package flagkit_test

import (
	"testing"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/flagkit"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Quiet tests ---

func TestQuiet_DefaultFalse(t *testing.T) {
	opts := &flagkit.Quiet{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.False(t, opts.Enabled)
}

func TestQuiet_ExplicitTrue(t *testing.T) {
	opts := &flagkit.Quiet{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--quiet"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.True(t, opts.Enabled)
}

func TestQuiet_ShortFlag(t *testing.T) {
	opts := &flagkit.Quiet{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"-q"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.True(t, opts.Enabled)
}

func TestQuiet_ExplicitFalse(t *testing.T) {
	opts := &flagkit.Quiet{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--quiet=false"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.False(t, opts.Enabled)
}

func TestQuiet_Standalone(t *testing.T) {
	opts := &flagkit.Quiet{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("quiet")
	require.NotNil(t, f, "flag should be registered")
	assert.Equal(t, "q", f.Shorthand)
	assert.Equal(t, "false", f.DefValue)
	assert.Equal(t, "Suppress non-essential output", f.Usage)
}

func TestQuiet_Annotation(t *testing.T) {
	opts := &flagkit.Quiet{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("quiet")
	require.NotNil(t, f)
	ann, ok := f.Annotations[flagkit.FlagKitAnnotation]
	assert.True(t, ok, "flagkit annotation should be set")
	assert.Equal(t, []string{"true"}, ann)
}

func TestQuiet_Attach_ErrorOnDuplicate(t *testing.T) {
	opts := &flagkit.Quiet{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	err := opts.Attach(cmd)
	assert.Error(t, err)
}

func TestQuiet_Embedded(t *testing.T) {
	type buildOpts struct {
		flagkit.Quiet
		Target string `flag:"target" flagdescr:"Build target" default:"all"`
	}
	opts := &buildOpts{}
	cmd := &cobra.Command{Use: "build"}
	require.NoError(t, structcli.Define(cmd, opts))

	require.NoError(t, cmd.Flags().Parse([]string{"-q", "--target", "linux"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.True(t, opts.Quiet.Enabled)
	assert.Equal(t, "linux", opts.Target)
}

func TestQuiet_JSONSchema(t *testing.T) {
	opts := &flagkit.Quiet{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	schemas, err := structcli.JSONSchema(cmd)
	require.NoError(t, err)
	require.Len(t, schemas, 1)

	_, ok := schemas[0].Flags["quiet"]
	assert.True(t, ok, "JSON schema should include the quiet flag")
}
