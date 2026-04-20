package flagkit_test

import (
	"testing"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/flagkit"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- DryRun tests ---

func TestDryRun_DefaultFalse(t *testing.T) {
	opts := &flagkit.DryRun{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.False(t, opts.Enabled)
}

func TestDryRun_ExplicitTrue(t *testing.T) {
	opts := &flagkit.DryRun{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--dry-run"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.True(t, opts.Enabled)
}

func TestDryRun_ExplicitFalse(t *testing.T) {
	opts := &flagkit.DryRun{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--dry-run=false"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.False(t, opts.Enabled)
}

func TestDryRun_Standalone(t *testing.T) {
	opts := &flagkit.DryRun{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("dry-run")
	require.NotNil(t, f, "flag should be registered")
	assert.Equal(t, "", f.Shorthand)
	assert.Equal(t, "false", f.DefValue)
	assert.Equal(t, "Preview without making changes", f.Usage)
}

func TestDryRun_Annotation(t *testing.T) {
	opts := &flagkit.DryRun{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("dry-run")
	require.NotNil(t, f)
	ann, ok := f.Annotations[flagkit.FlagKitAnnotation]
	assert.True(t, ok, "flagkit annotation should be set")
	assert.Equal(t, []string{"true"}, ann)
}

func TestDryRun_Attach_ErrorOnDuplicate(t *testing.T) {
	opts := &flagkit.DryRun{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	err := opts.Attach(cmd)
	assert.Error(t, err)
}

func TestDryRun_Embedded(t *testing.T) {
	type deployOpts struct {
		flagkit.DryRun
		Target string `flag:"target" flagdescr:"Deploy target" default:"staging"`
	}
	opts := &deployOpts{}
	cmd := &cobra.Command{Use: "deploy"}
	require.NoError(t, structcli.Define(cmd, opts))

	require.NoError(t, cmd.Flags().Parse([]string{"--dry-run", "--target", "prod"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.True(t, opts.DryRun.Enabled)
	assert.Equal(t, "prod", opts.Target)
}

func TestDryRun_JSONSchema(t *testing.T) {
	opts := &flagkit.DryRun{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	schemas, err := structcli.JSONSchema(cmd)
	require.NoError(t, err)
	require.Len(t, schemas, 1)

	_, ok := schemas[0].Flags["dry-run"]
	assert.True(t, ok, "JSON schema should include the dry-run flag")
}
