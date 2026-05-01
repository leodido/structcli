package structcli

import (
	"reflect"
	"testing"

	internalenv "github.com/leodido/structcli/internal/env"
	"github.com/leodido/structcli/values"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Option structs ---

type hiddenFlagOptions struct {
	Secret  string `flag:"secret" flaghidden:"true" flagdescr:"a secret flag"`
	Visible string `flag:"visible" flagdescr:"a visible flag"`
}

func (o *hiddenFlagOptions) Attach(c *cobra.Command) error { return nil }

type hiddenFalseFlagOptions struct {
	NotHidden string `flag:"not-hidden" flaghidden:"false" flagdescr:"explicitly not hidden"`
}

func (o *hiddenFalseFlagOptions) Attach(c *cobra.Command) error { return nil }

type hiddenEmptyFlagOptions struct {
	NotHidden string `flag:"not-hidden" flaghidden:"" flagdescr:"empty flaghidden"`
}

func (o *hiddenEmptyFlagOptions) Attach(c *cobra.Command) error { return nil }

type hiddenNoTagOptions struct {
	Normal string `flag:"normal" flagdescr:"no flaghidden tag"`
}

func (o *hiddenNoTagOptions) Attach(c *cobra.Command) error { return nil }

type hiddenPresetOptions struct {
	Level int `flag:"level" flaghidden:"true" flagpreset:"max=10" flagdescr:"level flag"`
}

func (o *hiddenPresetOptions) Attach(c *cobra.Command) error { return nil }

type hiddenMultiPresetOptions struct {
	Level int `flag:"level" flaghidden:"true" flagpreset:"low=1;high=9" flagdescr:"level flag"`
}

func (o *hiddenMultiPresetOptions) Attach(c *cobra.Command) error { return nil }

type notHiddenPresetOptions struct {
	Level int `flag:"level" flaghidden:"false" flagpreset:"max=10" flagdescr:"level flag"`
}

func (o *notHiddenPresetOptions) Attach(c *cobra.Command) error { return nil }

type hiddenShortOptions struct {
	Secret string `flag:"secret" flaghidden:"true" flagshort:"x" flagdescr:"hidden with short"`
}

func (o *hiddenShortOptions) Attach(c *cobra.Command) error { return nil }

type hiddenDefaultOptions struct {
	Port int `flag:"port" flaghidden:"true" default:"42" flagdescr:"hidden with default"`
}

func (o *hiddenDefaultOptions) Attach(c *cobra.Command) error { return nil }

type hiddenRequiredOptions struct {
	Token string `flag:"token" flaghidden:"true" flagrequired:"true" flagdescr:"hidden and required"`
}

func (o *hiddenRequiredOptions) Attach(c *cobra.Command) error { return nil }

type hiddenEnvOptions struct {
	Secret string `flag:"secret" flaghidden:"true" flagenv:"true" flagdescr:"hidden with env"`
}

func (o *hiddenEnvOptions) Attach(c *cobra.Command) error { return nil }

type hiddenGroupOptions struct {
	Secret string `flag:"secret" flaghidden:"true" flaggroup:"Advanced" flagdescr:"hidden with group"`
}

func (o *hiddenGroupOptions) Attach(c *cobra.Command) error { return nil }

type hiddenCustomOptions struct {
	Mode string `flag:"mode" flaghidden:"true" flagdescr:"hidden custom"`
}

func (o *hiddenCustomOptions) FieldHooks() map[string]FieldHook {
	return map[string]FieldHook{
		"Mode": {
			Define: func(name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
				fieldPtr := fieldValue.Addr().Interface().(*string)
				*fieldPtr = "default"

				return values.NewString(fieldPtr), descr
			},
			Decode: func(input any) (any, error) {
				return input, nil
			},
		},
	}
}

func (o *hiddenCustomOptions) Attach(c *cobra.Command) error { return nil }

// Validation error structs

type hiddenInvalidBoolOptions struct {
	Bad string `flag:"bad" flaghidden:"yes" flagdescr:"invalid bool"`
}

func (o *hiddenInvalidBoolOptions) Attach(c *cobra.Command) error { return nil }

type hiddenInvalidNumericOptions struct {
	Bad string `flag:"bad" flaghidden:"2" flagdescr:"invalid numeric"`
}

func (o *hiddenInvalidNumericOptions) Attach(c *cobra.Command) error { return nil }

type hiddenOnStructOptions struct {
	Nested hiddenNestedStruct `flaghidden:"true"`
}

type hiddenNestedStruct struct {
	Field string `flag:"field" flagdescr:"nested field"`
}

func (o *hiddenOnStructOptions) Attach(c *cobra.Command) error { return nil }

type hiddenIgnoreConflictOptions struct {
	Bad string `flag:"bad" flaghidden:"true" flagignore:"true" flagdescr:"conflict"`
}

func (o *hiddenIgnoreConflictOptions) Attach(c *cobra.Command) error { return nil }

type hiddenCaseTrueOptions struct {
	Secret string `flag:"secret" flaghidden:"True" flagdescr:"case true"`
}

func (o *hiddenCaseTrueOptions) Attach(c *cobra.Command) error { return nil }

type hiddenCaseFalseOptions struct {
	Visible string `flag:"visible" flaghidden:"FALSE" flagdescr:"case false"`
}

func (o *hiddenCaseFalseOptions) Attach(c *cobra.Command) error { return nil }

type hiddenNumericOneOptions struct {
	Secret string `flag:"secret" flaghidden:"1" flagdescr:"numeric 1"`
}

func (o *hiddenNumericOneOptions) Attach(c *cobra.Command) error { return nil }

type hiddenNumericZeroOptions struct {
	Visible string `flag:"visible" flaghidden:"0" flagdescr:"numeric 0"`
}

func (o *hiddenNumericZeroOptions) Attach(c *cobra.Command) error { return nil }

// --- Helpers ---

func resetFlagHiddenTestState() {
	viper.Reset()
	Reset()
	SetEnvPrefix("")
}

// --- Validation tests ---

func TestDefine_FlagHidden_InvalidBoolValue(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	err := Define(cmd, &hiddenInvalidBoolOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "flaghidden")
	assert.Contains(t, err.Error(), "Bad", "error should reference the field name")
}

func TestDefine_FlagHidden_InvalidNumericBoolValue(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	err := Define(cmd, &hiddenInvalidNumericOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "flaghidden")
	assert.Contains(t, err.Error(), "Bad", "error should reference the field name")
}

func TestDefine_FlagHidden_OnStructField(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	err := Define(cmd, &hiddenOnStructOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "flaghidden")
	assert.Contains(t, err.Error(), "struct")
	assert.Contains(t, err.Error(), "Nested", "error should reference the field name")
}

func TestDefine_FlagHidden_ConflictsWithFlagIgnore(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	err := Define(cmd, &hiddenIgnoreConflictOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "flagignore")
	assert.Contains(t, err.Error(), "flaghidden")
	assert.Contains(t, err.Error(), "Bad", "error should reference the field name")
}

func TestDefine_FlagHidden_ValidTrueValue(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	err := Define(cmd, &hiddenFlagOptions{})
	require.NoError(t, err)
}

func TestDefine_FlagHidden_ValidFalseValue(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	err := Define(cmd, &hiddenFalseFlagOptions{})
	require.NoError(t, err)
}

func TestDefine_FlagHidden_EmptyValue(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	err := Define(cmd, &hiddenEmptyFlagOptions{})
	require.NoError(t, err)
}

func TestDefine_FlagHidden_CaseInsensitiveTrue(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	err := Define(cmd, &hiddenCaseTrueOptions{})
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("secret")
	require.NotNil(t, flag)
	assert.True(t, flag.Hidden)
}

func TestDefine_FlagHidden_CaseInsensitiveFalse(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	err := Define(cmd, &hiddenCaseFalseOptions{})
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("visible")
	require.NotNil(t, flag)
	assert.False(t, flag.Hidden)
}

func TestDefine_FlagHidden_NumericOne(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	err := Define(cmd, &hiddenNumericOneOptions{})
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("secret")
	require.NotNil(t, flag)
	assert.True(t, flag.Hidden)
}

func TestDefine_FlagHidden_NumericZero(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	err := Define(cmd, &hiddenNumericZeroOptions{})
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("visible")
	require.NotNil(t, flag)
	assert.False(t, flag.Hidden)
}

// --- Define path tests ---

func TestDefine_FlagHidden_BasicHiddenFlag(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	err := Define(cmd, &hiddenFlagOptions{})
	require.NoError(t, err)

	secretFlag := cmd.Flags().Lookup("secret")
	require.NotNil(t, secretFlag)
	assert.True(t, secretFlag.Hidden, "flaghidden:'true' should set Hidden=true")

	visibleFlag := cmd.Flags().Lookup("visible")
	require.NotNil(t, visibleFlag)
	assert.False(t, visibleFlag.Hidden, "flag without flaghidden should have Hidden=false")
}

func TestDefine_FlagHidden_FalseKeepsFlagVisible(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	err := Define(cmd, &hiddenFalseFlagOptions{})
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("not-hidden")
	require.NotNil(t, flag)
	assert.False(t, flag.Hidden)
}

func TestDefine_FlagHidden_NoTagKeepsFlagVisible(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	err := Define(cmd, &hiddenNoTagOptions{})
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("normal")
	require.NotNil(t, flag)
	assert.False(t, flag.Hidden)
}

func TestDefine_FlagHidden_WithPreset(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &hiddenPresetOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)

	mainFlag := cmd.Flags().Lookup("level")
	require.NotNil(t, mainFlag)
	assert.True(t, mainFlag.Hidden, "main flag should be hidden")

	aliasFlag := cmd.Flags().Lookup("max")
	require.NotNil(t, aliasFlag)
	assert.True(t, aliasFlag.Hidden, "preset alias should inherit hidden from main flag")
}

func TestDefine_FlagHidden_WithMultiplePresets(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &hiddenMultiPresetOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)

	mainFlag := cmd.Flags().Lookup("level")
	require.NotNil(t, mainFlag)
	assert.True(t, mainFlag.Hidden)

	lowFlag := cmd.Flags().Lookup("low")
	require.NotNil(t, lowFlag)
	assert.True(t, lowFlag.Hidden, "preset alias 'low' should be hidden")

	highFlag := cmd.Flags().Lookup("high")
	require.NotNil(t, highFlag)
	assert.True(t, highFlag.Hidden, "preset alias 'high' should be hidden")
}

func TestDefine_FlagHidden_NotHiddenWithPreset(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &notHiddenPresetOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)

	mainFlag := cmd.Flags().Lookup("level")
	require.NotNil(t, mainFlag)
	assert.False(t, mainFlag.Hidden)

	aliasFlag := cmd.Flags().Lookup("max")
	require.NotNil(t, aliasFlag)
	assert.False(t, aliasFlag.Hidden)
}

func TestDefine_FlagHidden_WithShorthand(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &hiddenShortOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("secret")
	require.NotNil(t, flag)
	assert.True(t, flag.Hidden)
	assert.Equal(t, "x", flag.Shorthand)

	// Shorthand still works for parsing
	require.NoError(t, cmd.Flags().Parse([]string{"-x", "val"}))
	assert.Equal(t, "val", flag.Value.String())
}

func TestDefine_FlagHidden_WithDefault(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &hiddenDefaultOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("port")
	require.NotNil(t, flag)
	assert.True(t, flag.Hidden)
	assert.Equal(t, "42", flag.DefValue)
}

func TestDefine_FlagHidden_WithRequired(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &hiddenRequiredOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("token")
	require.NotNil(t, flag)
	assert.True(t, flag.Hidden, "flag should be hidden")
	assert.Equal(t, []string{"true"}, flag.Annotations[cobra.BashCompOneRequiredFlag], "flag should be required")
}

func TestDefine_FlagHidden_WithEnv(t *testing.T) {
	resetFlagHiddenTestState()
	SetEnvPrefix("APP")

	cmd := &cobra.Command{Use: "app"}
	opts := &hiddenEnvOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("secret")
	require.NotNil(t, flag)
	assert.True(t, flag.Hidden)
	assert.NotEmpty(t, flag.Annotations[internalenv.FlagAnnotation], "env annotation should be set on hidden flag")
}

func TestDefine_FlagHidden_WithGroup(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &hiddenGroupOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("secret")
	require.NotNil(t, flag)
	assert.True(t, flag.Hidden)
}

func TestDefine_FlagHidden_WithCustomHook(t *testing.T) {
	resetFlagHiddenTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &hiddenCustomOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("mode")
	require.NotNil(t, flag)
	assert.True(t, flag.Hidden, "custom hook flag should be hidden")
}

// --- Functional / integration tests ---

func TestUnmarshal_FlagHidden_AcceptsCLIValue(t *testing.T) {
	resetFlagHiddenTestState()

	opts := &hiddenFlagOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{"--secret=mysecret", "--visible=hello"}))
	require.NoError(t, Unmarshal(cmd, opts))

	assert.Equal(t, "mysecret", opts.Secret)
	assert.Equal(t, "hello", opts.Visible)
}

func TestUnmarshal_FlagHidden_EnvBindingWorks(t *testing.T) {
	resetFlagHiddenTestState()
	SetEnvPrefix("APP")

	t.Setenv("APP_SECRET", "envvalue")

	opts := &hiddenEnvOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, Unmarshal(cmd, opts))

	assert.Equal(t, "envvalue", opts.Secret)
}

func TestDefine_FlagHidden_ExcludedFromUsage(t *testing.T) {
	resetFlagHiddenTestState()

	opts := &hiddenFlagOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))

	usage := cmd.UsageString()
	assert.NotContains(t, usage, "--secret", "hidden flag should not appear in usage")
	assert.Contains(t, usage, "--visible", "visible flag should appear in usage")
}

func TestDefine_FlagHidden_PresetExcludedFromUsage(t *testing.T) {
	resetFlagHiddenTestState()

	opts := &hiddenPresetOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))

	usage := cmd.UsageString()
	assert.NotContains(t, usage, "--level", "hidden flag should not appear in usage")
	assert.NotContains(t, usage, "--max", "hidden preset alias should not appear in usage")
}

func TestUnmarshal_FlagHidden_PresetStillWorks(t *testing.T) {
	resetFlagHiddenTestState()

	opts := &hiddenPresetOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{"--max"}))
	require.NoError(t, Unmarshal(cmd, opts))

	assert.Equal(t, 10, opts.Level, "hidden preset alias should still set the value")
}

func TestJSONSchema_FlagHidden_ExcludedFromSchema(t *testing.T) {
	resetFlagHiddenTestState()

	opts := &hiddenFlagOptions{}
	cmd := &cobra.Command{Use: "app", Short: "test app"}
	require.NoError(t, Define(cmd, opts))

	schemas, err := JSONSchema(cmd)
	require.NoError(t, err)
	require.Len(t, schemas, 1)

	_, hasSecret := schemas[0].Flags["secret"]
	assert.False(t, hasSecret, "hidden flag should be excluded from JSON schema")

	_, hasVisible := schemas[0].Flags["visible"]
	assert.True(t, hasVisible, "visible flag should be included in JSON schema")
}
