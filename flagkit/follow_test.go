package flagkit_test

import (
	"testing"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/flagkit"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Standalone tests (Follow used directly via Attach) ---

func TestFollow_DefaultFalse(t *testing.T) {
	opts := &flagkit.Follow{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.False(t, opts.Enabled)
}

func TestFollow_ExplicitTrue(t *testing.T) {
	opts := &flagkit.Follow{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--follow"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.True(t, opts.Enabled)
}

func TestFollow_ShortFlag(t *testing.T) {
	opts := &flagkit.Follow{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"-f"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.True(t, opts.Enabled)
}

func TestFollow_ExplicitFalse(t *testing.T) {
	opts := &flagkit.Follow{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--follow=false"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.False(t, opts.Enabled)
}

func TestFollow_Standalone(t *testing.T) {
	opts := &flagkit.Follow{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("follow")
	require.NotNil(t, f, "flag should be registered")
	assert.Equal(t, "f", f.Shorthand)
	assert.Equal(t, "false", f.DefValue)
	assert.Equal(t, "Stream output continuously", f.Usage)
}

func TestFollow_Standalone_Annotation(t *testing.T) {
	opts := &flagkit.Follow{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("follow")
	require.NotNil(t, f)
	ann, ok := f.Annotations[flagkit.FlagKitAnnotation]
	assert.True(t, ok, "flagkit annotation should be set")
	assert.Equal(t, []string{"true"}, ann)
}

// --- Embedded tests (Follow embedded in a parent struct) ---

type logOptions struct {
	flagkit.Follow
	Service string `flag:"service" flagdescr:"Service name"`
}

func (o *logOptions) Attach(c *cobra.Command) error {
	if err := structcli.Define(c, o); err != nil {
		return err
	}
	flagkit.AnnotateCommand(c)

	return nil
}

func TestFollow_Embedded(t *testing.T) {
	opts := &logOptions{}
	cmd := &cobra.Command{Use: "logs"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--follow", "--service", "api"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.True(t, opts.Follow.Enabled)
	assert.Equal(t, "api", opts.Service)
}

func TestFollow_Embedded_DefaultFalse(t *testing.T) {
	opts := &logOptions{}
	cmd := &cobra.Command{Use: "logs"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--service", "api"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.False(t, opts.Follow.Enabled)
	assert.Equal(t, "api", opts.Service)
}

func TestFollow_Embedded_Annotation(t *testing.T) {
	opts := &logOptions{}
	cmd := &cobra.Command{Use: "logs"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("follow")
	require.NotNil(t, f)
	ann, ok := f.Annotations[flagkit.FlagKitAnnotation]
	assert.True(t, ok, "flagkit annotation should be set on embedded usage")
	assert.Equal(t, []string{"true"}, ann)
}

func TestFollow_Embedded_BothFlagsExist(t *testing.T) {
	opts := &logOptions{}
	cmd := &cobra.Command{Use: "logs"}
	require.NoError(t, opts.Attach(cmd))

	assert.NotNil(t, cmd.Flags().Lookup("follow"), "--follow should exist")
	assert.NotNil(t, cmd.Flags().Lookup("service"), "--service should exist")
}

// --- JSON Schema test ---

func TestFollow_JSONSchema(t *testing.T) {
	opts := &flagkit.Follow{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	schemas, err := structcli.JSONSchema(cmd)
	require.NoError(t, err)
	require.Len(t, schemas, 1)

	_, ok := schemas[0].Flags["follow"]
	assert.True(t, ok, "JSON schema should include the follow flag")
}

// --- Error path ---

func TestFollow_Attach_ErrorOnDuplicate(t *testing.T) {
	opts := &flagkit.Follow{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	// Second Attach on the same command triggers a Define validation error
	// because the "follow" flag is already registered.
	err := opts.Attach(cmd)
	assert.Error(t, err)
}

// --- AnnotateCommand on command without flagkit flags ---

func TestAnnotateCommand_NoFlagKitFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "app"}
	cmd.Flags().Bool("other", false, "some other flag")

	// Should not panic when no flagkit flags exist
	flagkit.AnnotateCommand(cmd)

	f := cmd.Flags().Lookup("other")
	require.NotNil(t, f)
	_, ok := f.Annotations[flagkit.FlagKitAnnotation]
	assert.False(t, ok, "non-flagkit flag should not be annotated")
}
