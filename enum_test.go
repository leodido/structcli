package structcli

import (
	"encoding/json"
	"testing"

	internalhooks "github.com/leodido/structcli/internal/hooks"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test enum type ---

type testEnvironment string

const (
	testEnvDev     testEnvironment = "dev"
	testEnvStaging testEnvironment = "staging"
	testEnvProd    testEnvironment = "prod"
)

func testEnvMap() map[testEnvironment][]string {
	return map[testEnvironment][]string{
		testEnvDev:     {"dev", "development"},
		testEnvStaging: {"staging", "stage"},
		testEnvProd:    {"prod", "production"},
	}
}

// --- Option structs ---

type enumOptions struct {
	Env testEnvironment `flag:"env" flagdescr:"target environment"`
}

func (o *enumOptions) Attach(c *cobra.Command) error { return nil }

type enumDefaultOptions struct {
	Env testEnvironment `flag:"env" flagdescr:"target environment" default:"staging"`
}

func (o *enumDefaultOptions) Attach(c *cobra.Command) error { return nil }

type enumRequiredOptions struct {
	Env testEnvironment `flag:"env" flagdescr:"target environment" flagrequired:"true"`
}

func (o *enumRequiredOptions) Attach(c *cobra.Command) error { return nil }

type enumEnvVarOptions struct {
	Env testEnvironment `flag:"env" flagdescr:"target environment" flagenv:"true"`
}

func (o *enumEnvVarOptions) Attach(c *cobra.Command) error { return nil }

type enumEnvOnlyOptions struct {
	Env testEnvironment `flag:"env" flagdescr:"target environment" flagenv:"only"`
}

func (o *enumEnvOnlyOptions) Attach(c *cobra.Command) error { return nil }

// --- Helpers ---

func resetEnumTestState() {
	viper.Reset()
	SetEnvPrefix("")
}

// saveAndRestoreRegistries saves all hook registries and restores them after
// the test. This prevents enum registration from leaking between tests.
func saveAndRestoreRegistries(t *testing.T) {
	t.Helper()

	origDefine := make(map[string]internalhooks.DefineHookFunc)
	for k, v := range internalhooks.DefineHookRegistry {
		origDefine[k] = v
	}

	decodeSnap := internalhooks.SnapshotDecodeRegistries()

	t.Cleanup(func() {
		internalhooks.DefineHookRegistry = origDefine
		internalhooks.RestoreDecodeRegistries(decodeSnap)
	})
}

func registerTestEnum(t *testing.T) {
	t.Helper()
	saveAndRestoreRegistries(t)
	RegisterEnum[testEnvironment](testEnvMap())
}

// --- Tests ---

func TestRegisterEnum_DefineAndUnmarshal(t *testing.T) {
	resetEnumTestState()
	registerTestEnum(t)

	opts := &enumOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{"--env", "prod"}))
	require.NoError(t, Unmarshal(cmd, opts))

	assert.Equal(t, testEnvProd, opts.Env)
}

func TestRegisterEnum_DefaultValue(t *testing.T) {
	resetEnumTestState()
	registerTestEnum(t)

	opts := &enumDefaultOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, Unmarshal(cmd, opts))

	assert.Equal(t, testEnvStaging, opts.Env)
}

func TestRegisterEnum_CaseInsensitive(t *testing.T) {
	resetEnumTestState()
	registerTestEnum(t)

	opts := &enumOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{"--env", "DEV"}))
	require.NoError(t, Unmarshal(cmd, opts))

	assert.Equal(t, testEnvDev, opts.Env)
}

func TestRegisterEnum_Aliases(t *testing.T) {
	resetEnumTestState()
	registerTestEnum(t)

	opts := &enumOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{"--env", "production"}))
	require.NoError(t, Unmarshal(cmd, opts))

	assert.Equal(t, testEnvProd, opts.Env)
}

func TestRegisterEnum_InvalidValue(t *testing.T) {
	resetEnumTestState()
	registerTestEnum(t)

	opts := &enumOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))

	err := cmd.Flags().Parse([]string{"--env", "invalid"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value")
}

func TestRegisterEnum_EnvVar(t *testing.T) {
	resetEnumTestState()
	registerTestEnum(t)
	SetEnvPrefix("APP")

	t.Setenv("APP_ENV", "staging")

	opts := &enumEnvVarOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, Unmarshal(cmd, opts))

	assert.Equal(t, testEnvStaging, opts.Env)
}

func TestRegisterEnum_EnvOnly(t *testing.T) {
	resetEnumTestState()
	registerTestEnum(t)
	SetEnvPrefix("APP")

	t.Setenv("APP_ENV", "prod")

	opts := &enumEnvOnlyOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))
	require.NoError(t, cmd.Flags().Parse([]string{}))
	require.NoError(t, Unmarshal(cmd, opts))

	assert.Equal(t, testEnvProd, opts.Env)
}

func TestRegisterEnum_NoFlagcustomNeeded(t *testing.T) {
	resetEnumTestState()
	registerTestEnum(t)

	opts := &enumOptions{}
	cmd := &cobra.Command{Use: "app"}
	// Define succeeds without flagcustom:"true"
	require.NoError(t, Define(cmd, opts))

	// Flag exists and works
	flag := cmd.Flags().Lookup("env")
	require.NotNil(t, flag)
	assert.Equal(t, "string", flag.Value.Type())
}

func TestRegisterEnum_JSONSchema(t *testing.T) {
	resetEnumTestState()
	registerTestEnum(t)

	opts := &enumOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))

	schema, err := JSONSchema(cmd)
	require.NoError(t, err)

	data, err := json.Marshal(schema)
	require.NoError(t, err)

	schemaStr := string(data)
	assert.Contains(t, schemaStr, `"enum"`)
	assert.Contains(t, schemaStr, `"dev"`)
	assert.Contains(t, schemaStr, `"staging"`)
	assert.Contains(t, schemaStr, `"prod"`)
}

func TestRegisterEnum_AutoCompletion(t *testing.T) {
	resetEnumTestState()
	registerTestEnum(t)

	opts := &enumOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))

	completionFunc, exists := cmd.GetFlagCompletionFunc("env")
	require.True(t, exists, "completion function should be auto-registered")

	suggestions, directive := completionFunc(cmd, nil, "")
	assert.Equal(t, []string{"dev", "prod", "staging"}, suggestions)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestRegisterEnum_DuplicatePanics(t *testing.T) {
	saveAndRestoreRegistries(t)

	RegisterEnum[testEnvironment](testEnvMap())

	assert.PanicsWithValue(t,
		`structcli: RegisterEnum: type "structcli.testEnvironment" is already registered`,
		func() { RegisterEnum[testEnvironment](testEnvMap()) },
	)
}

func TestRegisterEnum_EmptyValuesPanics(t *testing.T) {
	saveAndRestoreRegistries(t)

	assert.PanicsWithValue(t,
		"structcli: RegisterEnum: values must not be empty",
		func() { RegisterEnum[testEnvironment](nil) },
	)
	assert.PanicsWithValue(t,
		"structcli: RegisterEnum: values must not be empty",
		func() { RegisterEnum[testEnvironment](map[testEnvironment][]string{}) },
	)
}

func TestRegisterEnum_DescriptionEnhanced(t *testing.T) {
	resetEnumTestState()
	registerTestEnum(t)

	opts := &enumOptions{}
	cmd := &cobra.Command{Use: "app"}
	require.NoError(t, Define(cmd, opts))

	flag := cmd.Flags().Lookup("env")
	require.NotNil(t, flag)
	assert.Contains(t, flag.Usage, "{dev,prod,staging}")
}
