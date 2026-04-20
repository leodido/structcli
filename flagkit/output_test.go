package flagkit_test

import (
	"testing"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/flagkit"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	flagkit.RegisterOutputFormats(
		flagkit.OutputJSON,
		flagkit.OutputJSONL,
		flagkit.OutputText,
		flagkit.OutputYAML,
	)
}

// --- Output tests ---

func TestOutput_DefaultText(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, flagkit.OutputText, opts.Format)
}

func TestOutput_SetJSON(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--output", "json"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, flagkit.OutputJSON, opts.Format)
}

func TestOutput_SetYAML(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"--output", "yaml"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, flagkit.OutputYAML, opts.Format)
}

func TestOutput_SetJSONL(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"-o", "jsonl"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, flagkit.OutputJSONL, opts.Format)
}

func TestOutput_ShortFlag(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))
	require.NoError(t, cmd.Flags().Parse([]string{"-o", "json"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, flagkit.OutputJSON, opts.Format)
}

func TestOutput_Standalone(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("output")
	require.NotNil(t, f, "flag should be registered")
	assert.Equal(t, "o", f.Shorthand)
	assert.Equal(t, "text", f.DefValue)
	assert.Contains(t, f.Usage, "Output format")
}

func TestOutput_Annotation(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	f := cmd.Flags().Lookup("output")
	require.NotNil(t, f)
	ann, ok := f.Annotations[flagkit.FlagKitAnnotation]
	assert.True(t, ok, "flagkit annotation should be set")
	assert.Equal(t, []string{"true"}, ann)
}

func TestOutput_Attach_ErrorOnDuplicate(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	err := opts.Attach(cmd)
	assert.Error(t, err)
}

func TestOutput_Embedded(t *testing.T) {
	type listOpts struct {
		flagkit.OutputFmt
		Limit int `flag:"limit" flagdescr:"Max results" default:"10"`
	}
	opts := &listOpts{}
	cmd := &cobra.Command{Use: "list"}
	require.NoError(t, structcli.Define(cmd, opts))
	flagkit.AnnotateCommand(cmd)

	require.NoError(t, cmd.Flags().Parse([]string{"--output", "yaml", "--limit", "50"}))
	require.NoError(t, structcli.Unmarshal(cmd, opts))

	assert.Equal(t, flagkit.OutputYAML, opts.OutputFmt.Format)
	assert.Equal(t, 50, opts.Limit)
}

func TestOutput_JSONSchema(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	schemas, err := structcli.JSONSchema(cmd)
	require.NoError(t, err)
	require.Len(t, schemas, 1)

	_, ok := schemas[0].Flags["output"]
	assert.True(t, ok, "JSON schema should include the output flag")
}

// --- ValidFormat tests ---

func TestOutput_ValidFormat_Allowed(t *testing.T) {
	opts := &flagkit.OutputFmt{Format: flagkit.OutputJSON}
	err := opts.ValidFormat(flagkit.OutputJSON, flagkit.OutputText)
	assert.NoError(t, err)
}

func TestOutput_ValidFormat_NotAllowed(t *testing.T) {
	opts := &flagkit.OutputFmt{Format: flagkit.OutputYAML}
	err := opts.ValidFormat(flagkit.OutputJSON, flagkit.OutputText)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "yaml")
	assert.Contains(t, err.Error(), "allowed")
}

func TestOutput_ValidFormat_SingleAllowed(t *testing.T) {
	opts := &flagkit.OutputFmt{Format: flagkit.OutputText}
	assert.NoError(t, opts.ValidFormat(flagkit.OutputText))
}

func TestOutput_ValidFormat_NoRestriction(t *testing.T) {
	// No RestrictFormats called, no explicit args — all formats accepted.
	opts := &flagkit.OutputFmt{Format: flagkit.OutputYAML}
	assert.NoError(t, opts.ValidFormat())
}

func TestOutput_ValidFormat_UsesRestrictFormats(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	opts.RestrictFormats(cmd, flagkit.OutputJSON, flagkit.OutputText)

	// Set format to an allowed value — should pass with no args.
	opts.Format = flagkit.OutputJSON
	assert.NoError(t, opts.ValidFormat())

	// Set format to a disallowed value — should fail with no args.
	opts.Format = flagkit.OutputYAML
	err := opts.ValidFormat()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "yaml")
}

func TestOutput_ValidFormat_ExplicitOverridesStored(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	// Restrict to json+text via RestrictFormats.
	opts.RestrictFormats(cmd, flagkit.OutputJSON, flagkit.OutputText)

	// Explicit args override the stored set — yaml is allowed here.
	opts.Format = flagkit.OutputYAML
	assert.NoError(t, opts.ValidFormat(flagkit.OutputYAML, flagkit.OutputJSON))

	// But text is not in the explicit set.
	opts.Format = flagkit.OutputText
	err := opts.ValidFormat(flagkit.OutputYAML, flagkit.OutputJSON)
	assert.Error(t, err)
}

// --- RestrictFormats tests ---

func TestOutputFmt_RestrictFormats_Usage(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	opts.RestrictFormats(cmd, flagkit.OutputJSON, flagkit.OutputText)

	f := cmd.Flags().Lookup("output")
	require.NotNil(t, f)
	assert.Equal(t, "Output format {json,text}", f.Usage)
}

func TestOutputFmt_RestrictFormats_Annotation(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	opts.RestrictFormats(cmd, flagkit.OutputJSON, flagkit.OutputText)

	f := cmd.Flags().Lookup("output")
	require.NotNil(t, f)
	ann, ok := f.Annotations["___leodido_structcli_flagenum"]
	assert.True(t, ok, "enum annotation should be set")
	assert.Equal(t, []string{"json", "text"}, ann)
}

func TestOutputFmt_RestrictFormats_JSONSchema(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, opts.Attach(cmd))

	opts.RestrictFormats(cmd, flagkit.OutputJSON, flagkit.OutputText)

	schemas, err := structcli.JSONSchema(cmd)
	require.NoError(t, err)
	require.Len(t, schemas, 1)

	fs, ok := schemas[0].Flags["output"]
	require.True(t, ok)
	assert.Equal(t, []string{"json", "text"}, fs.Enum)
}

func TestOutputFmt_RestrictFormats_NoFlag(t *testing.T) {
	opts := &flagkit.OutputFmt{}
	cmd := &cobra.Command{Use: "app"}
	// Don't call Attach — no flag registered
	opts.RestrictFormats(cmd, flagkit.OutputJSON) // should not panic
}

// --- RegisterOutputFormats tests ---

func TestRegisterOutputFormats_PanicsOnDuplicate(t *testing.T) {
	// OutputFormat is already registered in init() above.
	// A second call should panic.
	assert.Panics(t, func() {
		flagkit.RegisterOutputFormats(flagkit.OutputJSON)
	})
}
