package flagkit_test

import (
	"testing"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/flagkit"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Verbose tests ---

func TestVerbose_DefaultZero(t *testing.T) {
	opts := &flagkit.Verbose{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, 0, opts.Level)
}

func TestVerbose_SingleV(t *testing.T) {
	opts := &flagkit.Verbose{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"-v"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, 1, opts.Level)
}

func TestVerbose_DoubleV(t *testing.T) {
	opts := &flagkit.Verbose{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"-v", "-v"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, 2, opts.Level)
}

func TestVerbose_LongFlag(t *testing.T) {
	opts := &flagkit.Verbose{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--verbose", "--verbose", "--verbose"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, 3, opts.Level)
}

func TestVerbose_Standalone(t *testing.T) {
	opts := &flagkit.Verbose{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("verbose")
	require.NotNil(t, f, "flag should be registered")
	assert.Equal(t, "v", f.Shorthand)
	assert.Contains(t, f.Usage, "Increase verbosity")
}

func TestVerbose_Annotation(t *testing.T) {
	opts := &flagkit.Verbose{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("verbose")
	require.NotNil(t, f)
	ann, ok := f.Annotations[flagkit.FlagKitAnnotation]
	assert.True(t, ok, "flagkit annotation should be set")
	assert.Equal(t, []string{"true"}, ann)
}

func TestVerbose_Attach_ErrorOnDuplicate(t *testing.T) {
	opts := &flagkit.Verbose{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	err := opts.Attach(cmd)
	assert.Error(t, err)
}

func TestVerbose_Embedded(t *testing.T) {
	type deployOpts struct {
		flagkit.Verbose
		Target string `flag:"target" flagdescr:"Deploy target" default:"staging"`
	}
	opts := &deployOpts{}
	cmd := &cobra.Command{Use: "deploy"}
	require.NoError(t, structcli.Define(cmd, opts))

	require.NoError(t, cmd.Flags().Parse([]string{"-v", "-v", "--target", "prod"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, 2, opts.Verbose.Level)
	assert.Equal(t, "prod", opts.Target)
}
