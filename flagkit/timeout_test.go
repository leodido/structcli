package flagkit_test

import (
	"testing"
	"time"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/flagkit"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Timeout tests ---

func TestTimeout_Default30s(t *testing.T) {
	opts := &flagkit.TimeoutOpt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, 30*time.Second, opts.Duration)
}

func TestTimeout_Set10s(t *testing.T) {
	opts := &flagkit.TimeoutOpt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--timeout", "10s"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, 10*time.Second, opts.Duration)
}

func TestTimeout_Set5m(t *testing.T) {
	opts := &flagkit.TimeoutOpt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--timeout", "5m"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, 5*time.Minute, opts.Duration)
}

func TestTimeout_Standalone(t *testing.T) {
	opts := &flagkit.TimeoutOpt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("timeout")
	require.NotNil(t, f, "flag should be registered")
	assert.Equal(t, "", f.Shorthand)
	assert.Equal(t, "30s", f.DefValue)
	assert.Equal(t, "Operation timeout", f.Usage)
}

func TestTimeout_Annotation(t *testing.T) {
	opts := &flagkit.TimeoutOpt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("timeout")
	require.NotNil(t, f)
	ann, ok := f.Annotations[flagkit.FlagKitAnnotation]
	assert.True(t, ok, "flagkit annotation should be set")
	assert.Equal(t, []string{"true"}, ann)
}

func TestTimeout_Attach_ErrorOnDuplicate(t *testing.T) {
	opts := &flagkit.TimeoutOpt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	err := opts.Attach(cmd)
	assert.Error(t, err)
}

func TestTimeout_Embedded(t *testing.T) {
	type fetchOpts struct {
		flagkit.TimeoutOpt
		URL string `flag:"url" flagdescr:"URL to fetch" flagrequired:"true"`
	}
	opts := &fetchOpts{}
	cmd := &cobra.Command{Use: "fetch"}
	require.NoError(t, structcli.Define(cmd, opts))

	require.NoError(t, cmd.Flags().Parse([]string{"--timeout", "2m", "--url", "https://example.com"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, 2*time.Minute, opts.TimeoutOpt.Duration)
	assert.Equal(t, "https://example.com", opts.URL)
}

func TestTimeout_JSONSchema(t *testing.T) {
	opts := &flagkit.TimeoutOpt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	schemas, err := structcli.JSONSchema(cmd)
	require.NoError(t, err)
	require.Len(t, schemas, 1)

	_, ok := schemas[0].Flags["timeout"]
	assert.True(t, ok, "JSON schema should include the timeout flag")
}
