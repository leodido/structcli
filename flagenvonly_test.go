package structcli

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	structclierrors "github.com/leodido/structcli/errors"
	internalenv "github.com/leodido/structcli/internal/env"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Option structs ---

type envOnlyOptions struct {
	Secret  string `flagenv:"only" flag:"secret" flagdescr:"a secret value"`
	Visible string `flag:"visible" flagdescr:"a visible flag"`
}

func (o *envOnlyOptions) Attach(c *cobra.Command) error { return nil }

type envOnlyRequiredOptions struct {
	APIKey string `flagenv:"only" flag:"api-key" flagrequired:"true" flagdescr:"API key"`
}

func (o *envOnlyRequiredOptions) Attach(c *cobra.Command) error { return nil }

type envOnlyDefaultOptions struct {
	Port string `flagenv:"only" flag:"port" default:"8080" flagdescr:"server port"`
}

func (o *envOnlyDefaultOptions) Attach(c *cobra.Command) error { return nil }

type envOnlyWithGroupOptions struct {
	Token string `flagenv:"only" flag:"token" flaggroup:"auth" flagdescr:"auth token"`
}

func (o *envOnlyWithGroupOptions) Attach(c *cobra.Command) error { return nil }

type envOnlyWithDescrOptions struct {
	Key string `flagenv:"only" flag:"key" flagdescr:"encryption key"`
}

func (o *envOnlyWithDescrOptions) Attach(c *cobra.Command) error { return nil }

// --- Invalid combinations ---

type envOnlyWithShortOptions struct {
	Bad string `flagenv:"only" flag:"bad" flagshort:"b"`
}

func (o *envOnlyWithShortOptions) Attach(c *cobra.Command) error { return nil }

type envOnlyWithPresetOptions struct {
	Bad string `flagenv:"only" flag:"bad" flagpreset:"max=10"`
}

func (o *envOnlyWithPresetOptions) Attach(c *cobra.Command) error { return nil }

type envOnlyWithTypeOptions struct {
	Bad int `flagenv:"only" flag:"bad" flagtype:"count"`
}

func (o *envOnlyWithTypeOptions) Attach(c *cobra.Command) error { return nil }

type envOnlyWithIgnoreOptions struct {
	Bad string `flagenv:"only" flagignore:"true"`
}

func (o *envOnlyWithIgnoreOptions) Attach(c *cobra.Command) error { return nil }

// --- Struct inheritance ---

type envOnlyStructOptions struct {
	Auth envOnlyStructAuth `flagenv:"only"`
}

type envOnlyStructAuth struct {
	User string `flag:"user" flagdescr:"username"`
	Pass string `flag:"pass" flagdescr:"password"`
}

func (o *envOnlyStructOptions) Attach(c *cobra.Command) error { return nil }

// --- Test helpers ---

func resetEnvOnlyTestState() {
	viper.Reset()
	SetEnvPrefix("")
}

// --- Validation tests ---

func TestDefine_EnvOnly_Accepted(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	cmd := &cobra.Command{Use: "app"}
	opts := &envOnlyOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)

	// Carrier flag exists and is hidden
	flag := cmd.Flags().Lookup("secret")
	require.NotNil(t, flag)
	assert.True(t, flag.Hidden, "env-only carrier flag should be hidden")

	// Has env-only annotation
	envOnlyAnnotation := flag.Annotations[internalenv.FlagEnvOnlyAnnotation]
	assert.Equal(t, []string{"true"}, envOnlyAnnotation)

	// Has env annotation
	envAnnotation := flag.Annotations[internalenv.FlagAnnotation]
	assert.NotEmpty(t, envAnnotation, "env-only flag should have env annotation")

	// Visible flag still works normally
	visibleFlag := cmd.Flags().Lookup("visible")
	require.NotNil(t, visibleFlag)
	assert.False(t, visibleFlag.Hidden)
}

func TestDefine_EnvOnly_WithDescr_Accepted(t *testing.T) {
	resetEnvOnlyTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &envOnlyWithDescrOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)
}

func TestDefine_EnvOnly_WithGroup_Accepted(t *testing.T) {
	resetEnvOnlyTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &envOnlyWithGroupOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)
}

func TestDefine_EnvOnly_WithRequired_Accepted(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	cmd := &cobra.Command{Use: "app"}
	opts := &envOnlyRequiredOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("api-key")
	require.NotNil(t, flag)

	// Should have cobra's standard required annotation (env-only uses MarkFlagRequired normally)
	reqAnnotation := flag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.Equal(t, []string{"true"}, reqAnnotation)
}

func TestDefine_EnvOnly_WithDefault_Accepted(t *testing.T) {
	resetEnvOnlyTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &envOnlyDefaultOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)
}

func TestDefine_EnvOnly_RejectsShort(t *testing.T) {
	resetEnvOnlyTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &envOnlyWithShortOptions{}
	err := Define(cmd, opts)
	require.Error(t, err)
	assert.ErrorIs(t, err, structclierrors.ErrConflictingTags)
	assert.Contains(t, err.Error(), "flagshort cannot be used with flagenv='only'")
}

func TestDefine_EnvOnly_RejectsPreset(t *testing.T) {
	resetEnvOnlyTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &envOnlyWithPresetOptions{}
	err := Define(cmd, opts)
	require.Error(t, err)
	assert.ErrorIs(t, err, structclierrors.ErrConflictingTags)
	assert.Contains(t, err.Error(), "flagpreset cannot be used with flagenv='only'")
}

func TestDefine_EnvOnly_RejectsType(t *testing.T) {
	resetEnvOnlyTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &envOnlyWithTypeOptions{}
	err := Define(cmd, opts)
	require.Error(t, err)
	assert.ErrorIs(t, err, structclierrors.ErrConflictingTags)
	assert.Contains(t, err.Error(), "flagtype cannot be used with flagenv='only'")
}

func TestDefine_EnvOnly_RejectsIgnore(t *testing.T) {
	resetEnvOnlyTestState()

	cmd := &cobra.Command{Use: "app"}
	opts := &envOnlyWithIgnoreOptions{}
	err := Define(cmd, opts)
	require.Error(t, err)
	assert.ErrorIs(t, err, structclierrors.ErrConflictingTags)
}

// --- Struct inheritance ---

func TestDefine_EnvOnly_StructInheritance(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	cmd := &cobra.Command{Use: "app"}
	opts := &envOnlyStructOptions{}
	err := Define(cmd, opts)
	require.NoError(t, err)

	// Children should have env binding (inherited from struct's flagenv:"only")
	userFlag := cmd.Flags().Lookup("user")
	require.NotNil(t, userFlag)
	assert.NotEmpty(t, userFlag.Annotations[internalenv.FlagAnnotation], "child should inherit env binding")
	// Children should NOT be env-only themselves (they get normal CLI flags)
	_, isEnvOnly := userFlag.Annotations[internalenv.FlagEnvOnlyAnnotation]
	assert.False(t, isEnvOnly, "child should not be env-only")
	assert.False(t, userFlag.Hidden, "child should be a normal visible flag")
}

// --- Unmarshal tests ---

func TestUnmarshal_EnvOnly_PopulatedFromEnv(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	t.Setenv("APP_SECRET", "mysecret")

	opts := &envOnlyOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, Unmarshal(cmd, opts))

	assert.Equal(t, "mysecret", opts.Secret)
}

func TestUnmarshal_EnvOnly_FallsBackToDefault(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	opts := &envOnlyDefaultOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, Unmarshal(cmd, opts))

	assert.Equal(t, "8080", opts.Port)
}

func TestUnmarshal_EnvOnly_RejectsCLIUsage(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	opts := &envOnlyOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))

	// pflag allows setting hidden flags, but Unmarshal rejects it for env-only fields
	require.NoError(t, cmd.Flags().Parse([]string{"--secret=shouldfail"}))

	err := Unmarshal(cmd, opts)
	require.Error(t, err)
	assert.ErrorIs(t, err, structclierrors.ErrEnvOnlyCLIUsage)
	assert.Contains(t, err.Error(), "secret")
	assert.Contains(t, err.Error(), "environment variable")
}

func TestHandleError_EnvOnlyCLIUsage(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	opts := &envOnlyOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{"--secret=bad"}))

	err := Unmarshal(cmd, opts)
	require.Error(t, err)

	code := HandleError(cmd, err, os.Stderr)
	assert.Equal(t, 11, code, "should use InvalidFlagValue exit code")
}

func TestUnmarshal_EnvOnly_CLINotSetPassesThrough(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	t.Setenv("APP_SECRET", "envvalue")

	opts := &envOnlyOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{"--visible=hello"}))
	require.NoError(t, Unmarshal(cmd, opts))

	assert.Equal(t, "envvalue", opts.Secret)
	assert.Equal(t, "hello", opts.Visible)
}

func TestUnmarshal_EnvOnly_RequiredMissing(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	opts := &envOnlyRequiredOptions{}
	cmd := &cobra.Command{
		Use:  "app",
		RunE: func(cmd *cobra.Command, args []string) error { return Unmarshal(cmd, opts) },
	}
	require.NoError(t, Define(cmd, opts))

	// Execute without setting the env var; cobra's required flag check fires.
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)
	// Cobra produces: required flag(s) "api-key" not set
	assert.Contains(t, err.Error(), "api-key")

	// HandleError should classify it as missing_required_env with exit code 26
	code := HandleError(cmd, err, os.Stderr)
	assert.Equal(t, 26, code, "should use EnvMissingRequired exit code")
}

func TestUnmarshal_EnvOnly_RequiredSatisfiedByEnv(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	t.Setenv("APP_API_KEY", "mykey")

	opts := &envOnlyRequiredOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, Unmarshal(cmd, opts))

	assert.Equal(t, "mykey", opts.APIKey)
}

// --- Help / usage tests ---

func TestDefine_EnvOnly_ExcludedFromHelp(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	opts := &envOnlyOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))

	usage := cmd.UsageString()
	assert.NotContains(t, usage, "--secret", "env-only flag should not appear in usage")
	assert.Contains(t, usage, "--visible", "visible flag should appear in usage")
}

// --- JSON schema tests ---

func TestJSONSchema_EnvOnly_IncludedWithMarker(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	opts := &envOnlyOptions{}
	cmd := &cobra.Command{Use: "app", Short: "test app"}
	require.NoError(t, Define(cmd, opts))

	schemas, err := JSONSchema(cmd)
	require.NoError(t, err)
	require.Len(t, schemas, 1)

	schema := schemas[0]

	// env-only field should be in schema
	secretFlag, ok := schema.Flags["secret"]
	require.True(t, ok, "env-only field should appear in JSON schema")
	assert.True(t, secretFlag.EnvOnly, "env-only field should have EnvOnly marker")
	assert.NotEmpty(t, secretFlag.EnvVars, "env-only field should have env vars in schema")

	// visible flag should also be there, without env-only
	visibleFlag, ok := schema.Flags["visible"]
	require.True(t, ok)
	assert.False(t, visibleFlag.EnvOnly)

	// Verify JSON serialization includes env_only
	data, err := json.Marshal(secretFlag)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"env_only":true`)
}

func TestJSONSchema_EnvOnly_Required(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	opts := &envOnlyRequiredOptions{}
	cmd := &cobra.Command{Use: "app", Short: "test app"}
	require.NoError(t, Define(cmd, opts))

	schemas, err := JSONSchema(cmd)
	require.NoError(t, err)
	require.Len(t, schemas, 1)

	flag, ok := schemas[0].Flags["api-key"]
	require.True(t, ok)
	assert.True(t, flag.EnvOnly)
	assert.True(t, flag.Required)
}

// --- HandleError tests ---

func TestHandleError_MissingRequiredEnv(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	opts := &envOnlyRequiredOptions{}
	cmd := &cobra.Command{
		Use:  "app",
		RunE: func(cmd *cobra.Command, args []string) error { return Unmarshal(cmd, opts) },
	}
	require.NoError(t, Define(cmd, opts))

	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)

	code := HandleError(cmd, err, os.Stderr)
	assert.Equal(t, 26, code, "should use EnvMissingRequired exit code")
}

// --- Lint warning tests ---

type lintHiddenEnvOptions struct {
	Secret string `flaghidden:"true" flagenv:"true" flag:"secret" flagdescr:"should trigger lint"`
}

func (o *lintHiddenEnvOptions) Attach(c *cobra.Command) error { return nil }

type lintHiddenEnvWithShortOptions struct {
	Secret string `flaghidden:"true" flagenv:"true" flag:"secret" flagshort:"s" flagdescr:"has short, no lint"`
}

func (o *lintHiddenEnvWithShortOptions) Attach(c *cobra.Command) error { return nil }

type lintHiddenEnvWithPresetOptions struct {
	Level int `flaghidden:"true" flagenv:"true" flag:"level" flagpreset:"quiet=0" flagdescr:"has preset, no lint"`
}

func (o *lintHiddenEnvWithPresetOptions) Attach(c *cobra.Command) error { return nil }

type lintHiddenNoEnvOptions struct {
	Secret string `flaghidden:"true" flag:"secret" flagdescr:"hidden but no env, no lint"`
}

func (o *lintHiddenNoEnvOptions) Attach(c *cobra.Command) error { return nil }

func TestDefine_Lint_HiddenEnvSuggestsEnvOnly(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	var stderr bytes.Buffer
	cmd := &cobra.Command{Use: "app"}
	cmd.SetErr(&stderr)
	opts := &lintHiddenEnvOptions{}
	require.NoError(t, Define(cmd, opts))

	output := stderr.String()
	assert.Contains(t, output, `flagenv:"only"`,
		"should suggest flagenv:\"only\" for flaghidden+flagenv")
	assert.Contains(t, output, "Secret")
}

func TestDefine_Lint_NoWarningWithShort(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	var stderr bytes.Buffer
	cmd := &cobra.Command{Use: "app"}
	cmd.SetErr(&stderr)
	opts := &lintHiddenEnvWithShortOptions{}
	require.NoError(t, Define(cmd, opts))

	assert.Empty(t, stderr.String(),
		"should not warn when flagshort is present (incompatible with flagenv:only)")
}

func TestDefine_Lint_NoWarningWithPreset(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	var stderr bytes.Buffer
	cmd := &cobra.Command{Use: "app"}
	cmd.SetErr(&stderr)
	opts := &lintHiddenEnvWithPresetOptions{}
	require.NoError(t, Define(cmd, opts))

	assert.Empty(t, stderr.String(),
		"should not warn when flagpreset is present (incompatible with flagenv:only)")
}

func TestDefine_Lint_NoWarningWithoutEnv(t *testing.T) {
	resetEnvOnlyTestState()

	var stderr bytes.Buffer
	cmd := &cobra.Command{Use: "app"}
	cmd.SetErr(&stderr)
	opts := &lintHiddenNoEnvOptions{}
	require.NoError(t, Define(cmd, opts))

	assert.Empty(t, stderr.String(),
		"should not warn when flagenv is not set")
}

type lintRedundantHiddenEnvOnlyOptions struct {
	Secret string `flaghidden:"true" flagenv:"only" flag:"secret" flagdescr:"redundant hidden with env-only"`
}

func (o *lintRedundantHiddenEnvOnlyOptions) Attach(c *cobra.Command) error { return nil }

func TestDefine_Lint_RedundantHiddenWithEnvOnly(t *testing.T) {
	resetEnvOnlyTestState()
	SetEnvPrefix("APP")

	var stderr bytes.Buffer
	cmd := &cobra.Command{Use: "app"}
	cmd.SetErr(&stderr)
	opts := &lintRedundantHiddenEnvOnlyOptions{}
	require.NoError(t, Define(cmd, opts))

	output := stderr.String()
	assert.Contains(t, output, "redundant",
		"should warn that flaghidden is redundant with flagenv:\"only\"")
	assert.Contains(t, output, "Secret")
}

