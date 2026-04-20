package structcli_test

import (
	"testing"
	"time"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// OutputFormat is a custom string enum type.
type OutputFormat string

const (
	OutputJSON OutputFormat = "json"
	OutputText OutputFormat = "text"
)

func init() {
	structcli.RegisterEnum(map[OutputFormat][]string{
		OutputJSON: {"json"},
		OutputText: {"text"},
	})
}

// Output is an embedded struct whose name collides with the flag name.
type Output struct {
	Format OutputFormat `flag:"output" flagdescr:"Output format" default:"text"`
}

type CollisionOpts struct {
	Output
	Limit int `flag:"limit" flagdescr:"Max results" default:"10"`
}

func (o *CollisionOpts) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

func TestUnmarshal_EmbeddedStructNameCollidesWithFlagName(t *testing.T) {
	opts := &CollisionOpts{}
	cmd := &cobra.Command{Use: "test", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	require.NoError(t, structcli.Define(cmd, opts))

	require.NoError(t, cmd.Flags().Parse([]string{"--output", "json", "--limit", "50"}))

	err := structcli.Unmarshal(cmd, opts)
	require.NoError(t, err, "Unmarshal should handle embedded struct name colliding with flag name")

	assert.Equal(t, OutputJSON, opts.Output.Format)
	assert.Equal(t, 50, opts.Limit)
}

func TestUnmarshal_EmbeddedStructNameCollidesWithFlagName_Default(t *testing.T) {
	opts := &CollisionOpts{}
	cmd := &cobra.Command{Use: "test", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	require.NoError(t, structcli.Define(cmd, opts))

	require.NoError(t, cmd.Flags().Parse([]string{}))

	err := structcli.Unmarshal(cmd, opts)
	require.NoError(t, err, "Unmarshal should handle embedded struct with default value")

	assert.Equal(t, OutputText, opts.Output.Format)
	assert.Equal(t, 10, opts.Limit)
}

// NonColliding uses a struct name that doesn't collide with flag name "output".
type NonCollidingOutput struct {
	Format OutputFormat `flag:"output" flagdescr:"Output format" default:"text"`
}

type NonCollidingOpts struct {
	NonCollidingOutput
	Limit int `flag:"limit" flagdescr:"Max results" default:"10"`
}

func (o *NonCollidingOpts) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

func TestUnmarshal_NonCollidingEmbeddedStruct_Default(t *testing.T) {
	opts := &NonCollidingOpts{}
	cmd := &cobra.Command{Use: "test", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	require.NoError(t, structcli.Define(cmd, opts))

	require.NoError(t, cmd.Flags().Parse([]string{}))

	err := structcli.Unmarshal(cmd, opts)
	require.NoError(t, err)

	assert.Equal(t, OutputText, opts.NonCollidingOutput.Format)
	assert.Equal(t, 10, opts.Limit)
}

func TestUnmarshal_NonCollidingEmbeddedStruct_Explicit(t *testing.T) {
	opts := &NonCollidingOpts{}
	cmd := &cobra.Command{Use: "test", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	require.NoError(t, structcli.Define(cmd, opts))

	require.NoError(t, cmd.Flags().Parse([]string{"--output", "json"}))

	err := structcli.Unmarshal(cmd, opts)
	require.NoError(t, err)

	assert.Equal(t, OutputJSON, opts.NonCollidingOutput.Format)
	assert.Equal(t, 10, opts.Limit)
}

// --- Config file tests for collision scenario ---
//
// Both flat (output: json) and nested (output: { format: json }) config
// forms work for embedded struct fields. The KeyRemappingHook restructures
// flat keys into the nested form before mapstructure decodes them.

func TestUnmarshal_EmbeddedCollision_ConfigNested(t *testing.T) {
	// Nested YAML form: output: { format: json }
	// This is the correct way to set embedded struct fields via config.
	opts := &CollisionOpts{}
	root := &cobra.Command{Use: "app"}
	cmd := &cobra.Command{Use: "test", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	root.AddCommand(cmd)
	require.NoError(t, structcli.Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{}))

	// Simulate a config file with nested output.format
	configVip := structcli.GetConfigViper(cmd)
	configVip.Set("test.output.format", "json")

	err := structcli.Unmarshal(cmd, opts)
	require.NoError(t, err)

	assert.Equal(t, OutputJSON, opts.Output.Format)
	assert.Equal(t, 10, opts.Limit)
}

func TestUnmarshal_EmbeddedCollision_ConfigFlat(t *testing.T) {
	// Flat YAML form: output: json
	// The KeyRemappingHook restructures this into the nested form
	// (output.format: json) before mapstructure decodes it.
	opts := &CollisionOpts{}
	root := &cobra.Command{Use: "app"}
	cmd := &cobra.Command{Use: "test", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	root.AddCommand(cmd)
	require.NoError(t, structcli.Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{}))

	// Simulate a config file with flat output key
	configVip := structcli.GetConfigViper(cmd)
	configVip.Set("test.output", "json")

	// The KeyRemappingHook restructures the flat key into the nested
	// form, so both flat and nested config keys work for embedded structs.
	err := structcli.Unmarshal(cmd, opts)
	require.NoError(t, err)

	assert.Equal(t, OutputJSON, opts.Output.Format)
	assert.Equal(t, 10, opts.Limit)
}

// Timeout collision: struct name matches flag name for a duration field.
type Timeout struct {
	Duration time.Duration `flag:"timeout" flagdescr:"Operation timeout" default:"30s"`
}

type TimeoutCollisionOpts struct {
	Timeout
	Retries int `flag:"retries" flagdescr:"Number of retries" default:"3"`
}

func (o *TimeoutCollisionOpts) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

func TestUnmarshal_EmbeddedTimeout_Default(t *testing.T) {
	opts := &TimeoutCollisionOpts{}
	cmd := &cobra.Command{Use: "test", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	require.NoError(t, structcli.Define(cmd, opts))

	require.NoError(t, cmd.Flags().Parse([]string{}))

	err := structcli.Unmarshal(cmd, opts)
	require.NoError(t, err)

	assert.Equal(t, 30*time.Second, opts.Timeout.Duration)
	assert.Equal(t, 3, opts.Retries)
}

func TestUnmarshal_EmbeddedTimeout_Explicit(t *testing.T) {
	opts := &TimeoutCollisionOpts{}
	cmd := &cobra.Command{Use: "test", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	require.NoError(t, structcli.Define(cmd, opts))

	require.NoError(t, cmd.Flags().Parse([]string{"--timeout", "5m"}))

	err := structcli.Unmarshal(cmd, opts)
	require.NoError(t, err)

	assert.Equal(t, 5*time.Minute, opts.Timeout.Duration)
	assert.Equal(t, 3, opts.Retries)
}
