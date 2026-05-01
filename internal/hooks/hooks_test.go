package internalhooks_test

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/leodido/structcli"
	internalenv "github.com/leodido/structcli/internal/env"
	internalhooks "github.com/leodido/structcli/internal/hooks"
	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zapcore"
)

type structcliSuite struct {
	suite.Suite
}

// unsetEnv is for tests that must observe a genuinely absent variable.
// t.Setenv can restore state, but it cannot temporarily remove a variable,
// and these cases distinguish unset from set-to-empty via os.LookupEnv.
func unsetEnv(t *testing.T, key string) {
	t.Helper()

	value, set := os.LookupEnv(key)
	t.Cleanup(func() {
		if set {
			_ = os.Setenv(key, value)
		} else {
			_ = os.Unsetenv(key)
		}
	})

	require.NoError(t, os.Unsetenv(key))
}

func TestStructCLISuite(t *testing.T) {
	suite.Run(t, new(structcliSuite))
}

func (suite *structcliSuite) SetupTest() {
	// Reset viper state before each test to prevent test pollution
	viper.Reset()
	// Reset global prefix
	structcli.SetEnvPrefix("")
}

// createTempYAMLFile creates a temporary YAML files for testing
func (suite *structcliSuite) createTempYAMLFile(content string) string {
	tmpFile, err := os.CreateTemp("", "structcli_test_*.yaml")
	require.NoError(suite.T(), err)

	_, err = tmpFile.WriteString(content)
	require.NoError(suite.T(), err)

	err = tmpFile.Close()
	require.NoError(suite.T(), err)

	return tmpFile.Name()
}

func loadConfigForCommand(t require.TestingT, cmd *cobra.Command, configFile string) {
	cfgVip := structcli.GetConfigViper(cmd)
	cfgVip.SetConfigFile(configFile)
	require.NoError(t, cfgVip.ReadInConfig())
}

func (suite *structcliSuite) TestStoreDecodeHookFunc() {
	// Create a command with a flag
	cmd := &cobra.Command{Use: "testcmd"}
	cmd.Flags().String("custom-field", "", "test flag")

	// Create a simple decode method
	decodeMethod := func(input any) (any, error) {
		return input, nil
	}
	decodeValue := reflect.ValueOf(decodeMethod)
	targetType := reflect.TypeOf("")

	// Call StoreDecodeHookFunc
	internalhooks.StoreDecodeHookFunc(cmd, "custom-field", decodeValue, targetType)

	// Assert the scope contains the custom decode hook with correct key
	scope := internalscope.Get(cmd)
	expectedKey := fmt.Sprintf("customDecodeHook_%s_%s", cmd.Name(), "custom-field")

	storedHook, exists := scope.GetCustomDecodeHook(expectedKey)
	require.True(suite.T(), exists, "Custom decode hook should be stored in scope")
	assert.NotNil(suite.T(), storedHook, "Stored hook should not be nil")

	// Assert the flag has the annotation with the correct key
	flag := cmd.Flags().Lookup("custom-field")
	require.NotNil(suite.T(), flag, "Flag should exist")

	annotations := flag.Annotations[internalhooks.FlagDecodeHookAnnotation]
	require.NotNil(suite.T(), annotations, "Flag should have decode hook annotation")
	require.Len(suite.T(), annotations, 1, "Should have exactly one annotation")
	assert.Equal(suite.T(), expectedKey, annotations[0], "Annotation should contain the correct key")
}

func (suite *structcliSuite) TestStoreCompletionHookFunc() {
	cmd := &cobra.Command{Use: "testcmd"}
	cmd.Flags().String("mode", "", "test flag")

	completeMethod := func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"dev", "prod"}, cobra.ShellCompDirectiveNoFileComp
	}

	internalhooks.StoreCompletionHookFunc(cmd, "mode", reflect.ValueOf(completeMethod))

	completion, exists := cmd.GetFlagCompletionFunc("mode")
	require.True(suite.T(), exists, "completion function should be registered")

	suggestions, directive := completion(cmd, nil, "")
	assert.Equal(suite.T(), []string{"dev", "prod"}, suggestions)
	assert.Equal(suite.T(), cobra.ShellCompDirectiveNoFileComp, directive)
}

func (suite *structcliSuite) TestStoreCompletionHookFunc_PanicsOnInvalidHookValue() {
	cmd := &cobra.Command{Use: "testcmd"}
	cmd.Flags().String("mode", "", "test flag")

	assert.PanicsWithValue(suite.T(),
		`structcli: invalid completion hook for flag "mode"`,
		func() { internalhooks.StoreCompletionHookFunc(cmd, "mode", reflect.Value{}) },
	)
}

func (suite *structcliSuite) TestStoreDecodeHookFuncDirect() {
	cmd := &cobra.Command{Use: "testcmd"}
	cmd.Flags().String("custom-field", "", "test flag")

	decodeFn := func(input any) (any, error) {
		return input, nil
	}
	targetType := reflect.TypeOf("")

	internalhooks.StoreDecodeHookFuncDirect(cmd, "custom-field", decodeFn, targetType)

	scope := internalscope.Get(cmd)
	expectedKey := fmt.Sprintf("customDecodeHook_%s_%s", cmd.Name(), "custom-field")

	storedHook, exists := scope.GetCustomDecodeHook(expectedKey)
	require.True(suite.T(), exists, "Custom decode hook should be stored in scope")
	assert.NotNil(suite.T(), storedHook, "Stored hook should not be nil")

	flag := cmd.Flags().Lookup("custom-field")
	require.NotNil(suite.T(), flag, "Flag should exist")

	annotations := flag.Annotations[internalhooks.FlagDecodeHookAnnotation]
	require.NotNil(suite.T(), annotations, "Flag should have decode hook annotation")
	require.Len(suite.T(), annotations, 1, "Should have exactly one annotation")
	assert.Equal(suite.T(), expectedKey, annotations[0], "Annotation should contain the correct key")
}

func (suite *structcliSuite) TestStoreCompletionHookFuncDirect() {
	cmd := &cobra.Command{Use: "testcmd"}
	cmd.Flags().String("mode", "", "test flag")

	completeFn := func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"dev", "prod"}, cobra.ShellCompDirectiveNoFileComp
	}

	internalhooks.StoreCompletionHookFuncDirect(cmd, "mode", completeFn)

	completion, exists := cmd.GetFlagCompletionFunc("mode")
	require.True(suite.T(), exists, "completion function should be registered")

	suggestions, directive := completion(cmd, nil, "")
	assert.Equal(suite.T(), []string{"dev", "prod"}, suggestions)
	assert.Equal(suite.T(), cobra.ShellCompDirectiveNoFileComp, directive)
}

type zapcoreLevelOptions struct {
	LogLevel zapcore.Level `default:"info" flagdescr:"the logging level" flagenv:"true"`
}

func (o *zapcoreLevelOptions) Attach(c *cobra.Command) error { return nil }

func (suite *structcliSuite) TestHooks_DefineZapcoreLevelFlag() {
	// Test just defining the flag, without config file
	opts := &zapcoreLevelOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := structcli.Define(cmd, opts)

	require.Nil(suite.T(), err)

	// Check if the flag was created
	flag := cmd.Flags().Lookup("loglevel")
	assert.NotNil(suite.T(), flag)
	assert.Contains(suite.T(), flag.Usage, "{debug,info,warn,error,dpanic,panic,fatal}")
}

func (suite *structcliSuite) TestHooks_ZapcoreLevelFromYAML() {
	// Create a temporary config file
	configContent := `loglevel: debug`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	// Define options with zapcore.Level field
	opts := &zapcoreLevelOptions{}

	cmd := &cobra.Command{Use: "test"}

	// Set up viper to read from our config file
	loadConfigForCommand(suite.T(), cmd, configFile)

	// Define flags and unmarshal
	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), zapcore.DebugLevel, opts.LogLevel)
}

type durationOptions struct {
	Timeout time.Duration `flag:"timeout" flagdescr:"request timeout" default:"30s"`
}

func (o *durationOptions) Attach(c *cobra.Command) error { return nil }

func (suite *structcliSuite) TestHooks_DurationFromFlag() {
	// Test setting duration via command line flag
	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	structcli.Define(cmd, opts)

	// Set flag value
	err := cmd.Flags().Set("timeout", "45s")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 45*time.Second, opts.Timeout)
}

func (suite *structcliSuite) TestHooks_DurationFromYAMLString() {
	// Test duration from YAML string format
	configContent := `timeout: "2m30s"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	expected := 2*time.Minute + 30*time.Second
	assert.Equal(suite.T(), expected, opts.Timeout)
}

func (suite *structcliSuite) TestHooks_DurationFromYAMLNumber() {
	// Test duration from YAML number (nanoseconds)
	configContent := `timeout: 5000000000` // 5 seconds in nanoseconds
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 5*time.Second, opts.Timeout)
}

func (suite *structcliSuite) TestHooks_DurationVariousFormats() {
	testCases := []struct {
		name     string
		yaml     string
		expected time.Duration
	}{
		{"milliseconds", `timeout: "500ms"`, 500 * time.Millisecond},
		{"seconds", `timeout: "30s"`, 30 * time.Second},
		{"minutes", `timeout: "5m"`, 5 * time.Minute},
		{"hours", `timeout: "2h"`, 2 * time.Hour},
		{"complex", `timeout: "1h30m45s"`, time.Hour + 30*time.Minute + 45*time.Second},
		{"microseconds", `timeout: "100us"`, 100 * time.Microsecond},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			configFile := suite.createTempYAMLFile(tc.yaml)
			defer os.Remove(configFile)

			opts := &durationOptions{}
			cmd := &cobra.Command{Use: "test"}

			loadConfigForCommand(t, cmd, configFile)

			structcli.Define(cmd, opts)
			err := structcli.Unmarshal(cmd, opts)

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, opts.Timeout)
		})
	}
}

func (suite *structcliSuite) TestHooks_DurationDefault() {
	// Test that default value is used when no config provided
	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 30*time.Second, opts.Timeout) // from default:"30s"
}

func (suite *structcliSuite) TestHooks_DurationFlagOverridesConfig() {
	// Test flag precedence over config
	configContent := `timeout: "1m"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)

	// Set flag value (should override config)
	err := cmd.Flags().Set("timeout", "90s")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 90*time.Second, opts.Timeout) // flag wins over config
}

type stringSliceOptions struct {
	Cgroups []string `flag:"cgroups" flagdescr:"list of cgroups to monitor" flagenv:"true"`
}

func (o *stringSliceOptions) Attach(c *cobra.Command) error { return nil }

func flagEnvVarName(t testing.TB, cmd *cobra.Command, flagName string) string {
	t.Helper()

	flag := cmd.Flags().Lookup(flagName)
	require.NotNil(t, flag)

	envAnnotation := flag.Annotations[internalenv.FlagAnnotation]
	require.NotEmpty(t, envAnnotation)

	return envAnnotation[0]
}

func (suite *structcliSuite) TestHooks_StringSliceFromFlag() {
	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	structcli.Define(cmd, opts)

	err := cmd.Flags().Set("cgroups", "group1,group2,group3")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group1", "group2", "group3"}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceFromConfigCSV() {
	configContent := `cgroups: "group1,group2,group3"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group1", "group2", "group3"}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceFromConfigYAMLArray() {
	configContent := `cgroups:
  - group1
  - group2
  - group3`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group1", "group2", "group3"}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceEmptyString() {
	configContent := `cgroups: ""`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceQuotedComma() {
	configContent := `cgroups: '"group,1",group2'`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group,1", "group2"}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceEmptyEntries() {
	configContent := `cgroups: ",group1,,group2,"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"", "group1", "", "group2", ""}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceMultipleFlags() {
	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	structcli.Define(cmd, opts)

	// Set flag multiple times
	err := cmd.Flags().Set("cgroups", "group1")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("cgroups", "group2")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("cgroups", "group3")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group1", "group2", "group3"}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceFromEnv() {
	structcli.SetEnvPrefix("CSVAPP")
	defer structcli.SetEnvPrefix("")

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	structcli.Define(cmd, opts)

	envVarName := flagEnvVarName(suite.T(), cmd, "cgroups")
	suite.T().Setenv(envVarName, "env1,env2,env3")

	err := structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"env1", "env2", "env3"}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceFlagOverridesEnvAndConfig() {
	configContent := `cgroups: "config1,config2"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	structcli.SetEnvPrefix("CSVAPP")
	defer structcli.SetEnvPrefix("")

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)

	envVarName := flagEnvVarName(suite.T(), cmd, "cgroups")
	suite.T().Setenv(envVarName, "env1,env2")

	err := cmd.Flags().Set("cgroups", "flag1,flag2,flag3")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"flag1", "flag2", "flag3"}, opts.Cgroups)
}

type intSliceOptions struct {
	Ports []int `flag:"ports" flagdescr:"list of ports to listen on"`
}

func (o *intSliceOptions) Attach(c *cobra.Command) error { return nil }

func (suite *structcliSuite) TestHooks_IntSliceFromFlag() {
	// Test setting int slice via command line flag
	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	structcli.Define(cmd, opts)

	// Set flag value (simulating command line)
	err := cmd.Flags().Set("ports", "8080,9090,3000")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceFromYAMLCommaSeparated() {
	// Test hook converting comma-separated string from YAML to []int
	configContent := `ports: "8080,9090,3000"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceFromYAMLArray() {
	// Test YAML array directly (no hook needed, mapstructure handles this)
	configContent := `ports:
  - 8080
  - 9090
  - 3000`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceEmptyString() {
	// Test hook behavior with empty string
	configContent := `ports: ""`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	// StringToIntSliceHookFunc with empty string results in empty slice
	assert.Equal(suite.T(), []int{}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceSingleValue() {
	// Test hook with single value (no commas)
	configContent := `ports: "8080"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceWithSpaces() {
	// Test hook with values containing spaces (should be trimmed)
	configContent := `ports: " 8080 , 9090 , 3000 "`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceNegativeNumbers() {
	// Test hook with negative numbers
	configContent := `ports: "-1,0,8080,-9090"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{-1, 0, 8080, -9090}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceMultipleFlags() {
	// Test setting multiple flag values
	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	structcli.Define(cmd, opts)

	// Set flag multiple times
	err := cmd.Flags().Set("ports", "8080")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("ports", "9090")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("ports", "3000")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceFlagOverridesConfig() {
	// Test that flag values override config values
	configContent := `ports: "8080,9090"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)

	// Set flag value (should override config)
	err := cmd.Flags().Set("ports", "3000,4000,5000")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	// Flag values should win over config values
	assert.Equal(suite.T(), []int{3000, 4000, 5000}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceInvalidInteger() {
	// Test hook error handling with invalid integer
	configContent := `ports: "8080,invalid,3000"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid integer")
	assert.Contains(suite.T(), err.Error(), "invalid")
	assert.Contains(suite.T(), err.Error(), "couldn't unmarshal config to options:")
}

func (suite *structcliSuite) TestHooks_IntSliceFloatNumber() {
	// Test hook error handling with float number
	configContent := `ports: "8080,90.5,3000"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid integer")
	assert.Contains(suite.T(), err.Error(), "couldn't unmarshal config to options:")
	assert.Contains(suite.T(), err.Error(), "90.5")
}

func (suite *structcliSuite) TestHooks_IntSliceOutOfRange() {
	// Test hook error handling with number out of int range
	configContent := `ports: "8080,99999999999999999999,3000"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid integer")
	assert.Contains(suite.T(), err.Error(), "couldn't unmarshal config to options:")
}

type additionalSliceOptions struct {
	Durations []time.Duration `flag:"durations" flagdescr:"durations" flagenv:"true" default:"1s,2m"`
	Bools     []bool          `flag:"bools" flagdescr:"bools" flagenv:"true" default:"true,false"`
	Uints     []uint          `flag:"uints" flagdescr:"uints" flagenv:"true" default:"1,2,3"`
}

func (o *additionalSliceOptions) Attach(c *cobra.Command) error { return nil }

func (suite *structcliSuite) TestHooks_AdditionalSlicesDefaults() {
	opts := &additionalSliceOptions{}
	cmd := &cobra.Command{Use: "sliceapp"}

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []time.Duration{time.Second, 2 * time.Minute}, opts.Durations)
	assert.Equal(suite.T(), []bool{true, false}, opts.Bools)
	assert.Equal(suite.T(), []uint{1, 2, 3}, opts.Uints)
}

func (suite *structcliSuite) TestHooks_AdditionalSlicesFromFlag() {
	opts := &additionalSliceOptions{}
	cmd := &cobra.Command{Use: "sliceapp"}

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)

	err = cmd.Flags().Set("durations", "5s,1m30s")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("bools", "true,false,true")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("uints", "5,8,13")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []time.Duration{5 * time.Second, 90 * time.Second}, opts.Durations)
	assert.Equal(suite.T(), []bool{true, false, true}, opts.Bools)
	assert.Equal(suite.T(), []uint{5, 8, 13}, opts.Uints)
}

func (suite *structcliSuite) TestHooks_AdditionalSlicesMultipleFlags() {
	opts := &additionalSliceOptions{}
	cmd := &cobra.Command{Use: "sliceapp"}

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)

	err = cmd.Flags().Set("durations", "5s")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("durations", "1m")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("bools", "true")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("bools", "false")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("uints", "5")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("uints", "8")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []time.Duration{5 * time.Second, time.Minute}, opts.Durations)
	assert.Equal(suite.T(), []bool{true, false}, opts.Bools)
	assert.Equal(suite.T(), []uint{5, 8}, opts.Uints)
}

func (suite *structcliSuite) TestHooks_AdditionalSlicesFromConfigString() {
	configContent := `durations: "5s,1m30s"
bools: "true,false,true"
uints: "5,8,13"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &additionalSliceOptions{}
	cmd := &cobra.Command{Use: "sliceapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []time.Duration{5 * time.Second, 90 * time.Second}, opts.Durations)
	assert.Equal(suite.T(), []bool{true, false, true}, opts.Bools)
	assert.Equal(suite.T(), []uint{5, 8, 13}, opts.Uints)
}

func (suite *structcliSuite) TestHooks_AdditionalSlicesFromConfigYAMLArray() {
	configContent := `durations:
  - 5s
  - 1m30s
bools:
  - true
  - false
  - true
uints:
  - 5
  - 8
  - 13`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &additionalSliceOptions{}
	cmd := &cobra.Command{Use: "sliceapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []time.Duration{5 * time.Second, 90 * time.Second}, opts.Durations)
	assert.Equal(suite.T(), []bool{true, false, true}, opts.Bools)
	assert.Equal(suite.T(), []uint{5, 8, 13}, opts.Uints)
}

func (suite *structcliSuite) TestHooks_AdditionalSlicesFromEnv() {
	structcli.SetEnvPrefix("SLICEAPP")
	defer structcli.SetEnvPrefix("")

	opts := &additionalSliceOptions{}
	cmd := &cobra.Command{Use: "sliceapp"}

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)

	suite.T().Setenv(flagEnvVarName(suite.T(), cmd, "durations"), "5s,1m30s")
	suite.T().Setenv(flagEnvVarName(suite.T(), cmd, "bools"), "true,false,true")
	suite.T().Setenv(flagEnvVarName(suite.T(), cmd, "uints"), "5,8,13")

	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []time.Duration{5 * time.Second, 90 * time.Second}, opts.Durations)
	assert.Equal(suite.T(), []bool{true, false, true}, opts.Bools)
	assert.Equal(suite.T(), []uint{5, 8, 13}, opts.Uints)
}

func (suite *structcliSuite) TestHooks_AdditionalSlicesFlagOverridesEnvAndConfig() {
	configContent := `durations: "10s,20s"
bools: "false,false"
uints: "1,2"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	structcli.SetEnvPrefix("SLICEAPP")
	defer structcli.SetEnvPrefix("")

	opts := &additionalSliceOptions{}
	cmd := &cobra.Command{Use: "sliceapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)

	suite.T().Setenv(flagEnvVarName(suite.T(), cmd, "durations"), "30s,40s")
	suite.T().Setenv(flagEnvVarName(suite.T(), cmd, "bools"), "true,true")
	suite.T().Setenv(flagEnvVarName(suite.T(), cmd, "uints"), "3,4")

	err = cmd.Flags().Set("durations", "50s,1m")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("bools", "true,false,true")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("uints", "5,6,7")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []time.Duration{50 * time.Second, time.Minute}, opts.Durations)
	assert.Equal(suite.T(), []bool{true, false, true}, opts.Bools)
	assert.Equal(suite.T(), []uint{5, 6, 7}, opts.Uints)
}

func (suite *structcliSuite) TestHooks_AdditionalSlicesEmptyString() {
	configContent := `durations: ""
bools: ""
uints: ""`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &additionalSliceOptions{}
	cmd := &cobra.Command{Use: "sliceapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []time.Duration{}, opts.Durations)
	assert.Equal(suite.T(), []bool{}, opts.Bools)
	assert.Equal(suite.T(), []uint{}, opts.Uints)
}

func (suite *structcliSuite) TestHooks_DurationSliceInvalidValue() {
	configContent := `durations: "5s,nope"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &additionalSliceOptions{}
	cmd := &cobra.Command{Use: "sliceapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid string for []time.Duration")
	assert.Contains(suite.T(), err.Error(), "nope")
}

func (suite *structcliSuite) TestHooks_BoolSliceInvalidValue() {
	configContent := `bools: "true,maybe"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &additionalSliceOptions{}
	cmd := &cobra.Command{Use: "sliceapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid string for []bool")
	assert.Contains(suite.T(), err.Error(), "maybe")
}

func (suite *structcliSuite) TestHooks_UintSliceInvalidValue() {
	configContent := `uints: "1,nope"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &additionalSliceOptions{}
	cmd := &cobra.Command{Use: "sliceapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid string for []uint")
	assert.Contains(suite.T(), err.Error(), "nope")
}

func (suite *structcliSuite) TestHooks_UintSliceRejectsNegativeArrayValue() {
	configContent := `uints:
  - 1
  - -2`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &additionalSliceOptions{}
	cmd := &cobra.Command{Use: "sliceapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	require.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid negative uint")
	assert.Contains(suite.T(), err.Error(), "position 1")
}

type mapOptions struct {
	Labels map[string]string `flag:"labels" flagdescr:"labels" flagenv:"true" default:"env=dev,team=core"`
	Limits map[string]int    `flag:"limits" flagdescr:"limits" flagenv:"true" default:"cpu=2,memory=4"`
	Counts map[string]int64  `flag:"counts" flagdescr:"counts" flagenv:"true" default:"ok=1,fail=2"`
}

func (o *mapOptions) Attach(c *cobra.Command) error { return nil }

func (suite *structcliSuite) TestHooks_MapFamiliesDefaults() {
	opts := &mapOptions{}
	cmd := &cobra.Command{Use: "mapapp"}

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), map[string]string{"env": "dev", "team": "core"}, opts.Labels)
	assert.Equal(suite.T(), map[string]int{"cpu": 2, "memory": 4}, opts.Limits)
	assert.Equal(suite.T(), map[string]int64{"ok": 1, "fail": 2}, opts.Counts)
}

func (suite *structcliSuite) TestHooks_MapFamiliesFromFlag() {
	opts := &mapOptions{}
	cmd := &cobra.Command{Use: "mapapp"}

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)

	err = cmd.Flags().Set("labels", "env=prod,team=platform")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("limits", "cpu=8,memory=16")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("counts", "ok=10,fail=3")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), map[string]string{"env": "prod", "team": "platform"}, opts.Labels)
	assert.Equal(suite.T(), map[string]int{"cpu": 8, "memory": 16}, opts.Limits)
	assert.Equal(suite.T(), map[string]int64{"ok": 10, "fail": 3}, opts.Counts)
}

func (suite *structcliSuite) TestHooks_MapFamiliesFromConfigString() {
	configContent := `labels: "env=prod,team=platform"
limits: "cpu=8,memory=16"
counts: "ok=10,fail=3"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &mapOptions{}
	cmd := &cobra.Command{Use: "mapapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), map[string]string{"env": "prod", "team": "platform"}, opts.Labels)
	assert.Equal(suite.T(), map[string]int{"cpu": 8, "memory": 16}, opts.Limits)
	assert.Equal(suite.T(), map[string]int64{"ok": 10, "fail": 3}, opts.Counts)
}

func (suite *structcliSuite) TestHooks_MapFamiliesFromConfigYAMLMap() {
	configContent := `labels:
  env: prod
  team: platform
limits:
  cpu: 8
  memory: 16
counts:
  ok: 10
  fail: 3`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &mapOptions{}
	cmd := &cobra.Command{Use: "mapapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), map[string]string{"env": "prod", "team": "platform"}, opts.Labels)
	assert.Equal(suite.T(), map[string]int{"cpu": 8, "memory": 16}, opts.Limits)
	assert.Equal(suite.T(), map[string]int64{"ok": 10, "fail": 3}, opts.Counts)
}

func (suite *structcliSuite) TestHooks_MapFamiliesFromEnv() {
	structcli.SetEnvPrefix("MAPAPP")
	defer structcli.SetEnvPrefix("")

	opts := &mapOptions{}
	cmd := &cobra.Command{Use: "mapapp"}

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)

	suite.T().Setenv(flagEnvVarName(suite.T(), cmd, "labels"), "env=prod,team=platform")
	suite.T().Setenv(flagEnvVarName(suite.T(), cmd, "limits"), "cpu=8,memory=16")
	suite.T().Setenv(flagEnvVarName(suite.T(), cmd, "counts"), "ok=10,fail=3")

	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), map[string]string{"env": "prod", "team": "platform"}, opts.Labels)
	assert.Equal(suite.T(), map[string]int{"cpu": 8, "memory": 16}, opts.Limits)
	assert.Equal(suite.T(), map[string]int64{"ok": 10, "fail": 3}, opts.Counts)
}

func (suite *structcliSuite) TestHooks_MapFamiliesFlagOverridesEnvAndConfig() {
	configContent := `labels: "env=dev,team=core"
limits: "cpu=2,memory=4"
counts: "ok=1,fail=2"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	structcli.SetEnvPrefix("MAPAPP")
	defer structcli.SetEnvPrefix("")

	opts := &mapOptions{}
	cmd := &cobra.Command{Use: "mapapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)

	suite.T().Setenv(flagEnvVarName(suite.T(), cmd, "labels"), "env=stage,team=api")
	suite.T().Setenv(flagEnvVarName(suite.T(), cmd, "limits"), "cpu=4,memory=8")
	suite.T().Setenv(flagEnvVarName(suite.T(), cmd, "counts"), "ok=6,fail=1")

	err = cmd.Flags().Set("labels", "env=prod,team=platform")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("limits", "cpu=8,memory=16")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("counts", "ok=10,fail=3")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), map[string]string{"env": "prod", "team": "platform"}, opts.Labels)
	assert.Equal(suite.T(), map[string]int{"cpu": 8, "memory": 16}, opts.Limits)
	assert.Equal(suite.T(), map[string]int64{"ok": 10, "fail": 3}, opts.Counts)
}

func (suite *structcliSuite) TestHooks_MapFamiliesEmptyString() {
	configContent := `labels: ""
limits: ""
counts: ""`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &mapOptions{}
	cmd := &cobra.Command{Use: "mapapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), map[string]string{}, opts.Labels)
	assert.Equal(suite.T(), map[string]int{}, opts.Limits)
	assert.Equal(suite.T(), map[string]int64{}, opts.Counts)
}

func (suite *structcliSuite) TestHooks_MapFamiliesInvalidStringValue() {
	configContent := `limits: "cpu=fast,memory=16"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &mapOptions{}
	cmd := &cobra.Command{Use: "mapapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid string for map[string]int")
	assert.Contains(suite.T(), err.Error(), "cpu=fast")
}

func (suite *structcliSuite) TestHooks_MapFamiliesInvalidStringFormat() {
	configContent := `labels: "env"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &mapOptions{}
	cmd := &cobra.Command{Use: "mapapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid string for map[string]string")
	assert.Contains(suite.T(), err.Error(), "env")
}

func (suite *structcliSuite) TestHooks_MapFamiliesInvalidYAMLValue() {
	configContent := `counts:
  ok: nope`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &mapOptions{}
	cmd := &cobra.Command{Use: "mapapp"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	require.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "couldn't unmarshal config to options:")
	assert.Contains(suite.T(), err.Error(), "key \"ok\"")
}

type bytesOptions struct {
	Raw    []byte           `flag:"raw" flagdescr:"raw bytes" flagenv:"true" default:"raw-default"`
	Hex    structcli.Hex    `flag:"hex" flagdescr:"hex bytes" flagenv:"true" default:"68656c6c6f"`
	Base64 structcli.Base64 `flag:"base64" flagdescr:"base64 bytes" flagenv:"true" default:"aGVsbG8="`
}

func (o *bytesOptions) Attach(c *cobra.Command) error { return nil }

func (suite *structcliSuite) TestHooks_BytesDefaults() {
	opts := &bytesOptions{}
	cmd := &cobra.Command{Use: "bytes"}

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []byte("raw-default"), opts.Raw)
	assert.Equal(suite.T(), []byte("hello"), []byte(opts.Hex))
	assert.Equal(suite.T(), []byte("hello"), []byte(opts.Base64))
}

func (suite *structcliSuite) TestHooks_BytesFromFlag() {
	opts := &bytesOptions{}
	cmd := &cobra.Command{Use: "bytes"}

	structcli.Define(cmd, opts)

	err := cmd.Flags().Set("raw", "from-flag")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("hex", "666c61672d686578") // "flag-hex"
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("base64", "ZmxhZy1iYXNlNjQ=") // "flag-base64"
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []byte("from-flag"), opts.Raw)
	assert.Equal(suite.T(), []byte("flag-hex"), []byte(opts.Hex))
	assert.Equal(suite.T(), []byte("flag-base64"), []byte(opts.Base64))
}

func (suite *structcliSuite) TestHooks_BytesFromYAML() {
	configContent := `raw: "from-config"
hex: "636f6e6669672d686578"
base64: "Y29uZmlnLWJhc2U2NA=="`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &bytesOptions{}
	cmd := &cobra.Command{Use: "bytes"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []byte("from-config"), opts.Raw)
	assert.Equal(suite.T(), []byte("config-hex"), []byte(opts.Hex))
	assert.Equal(suite.T(), []byte("config-base64"), []byte(opts.Base64))
}

func (suite *structcliSuite) TestHooks_BytesFromEnv() {
	const (
		envRaw    = "BYTESAPP_RAW"
		envHex    = "BYTESAPP_HEX"
		envBase64 = "BYTESAPP_BASE64"
	)
	defer func() {
		structcli.SetEnvPrefix("")
	}()

	suite.T().Setenv(envRaw, "from-env")
	suite.T().Setenv(envHex, "656e762d686578")      // "env-hex"
	suite.T().Setenv(envBase64, "ZW52LWJhc2U2NA==") // "env-base64"
	structcli.SetEnvPrefix("bytesapp")

	opts := &bytesOptions{}
	cmd := &cobra.Command{Use: "bytesapp"}

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []byte("from-env"), opts.Raw)
	assert.Equal(suite.T(), []byte("env-hex"), []byte(opts.Hex))
	assert.Equal(suite.T(), []byte("env-base64"), []byte(opts.Base64))
}

func (suite *structcliSuite) TestHooks_BytesFlagOverridesConfig() {
	configContent := `raw: "config-raw"
hex: "636f6e6669672d686578"
base64: "Y29uZmlnLWJhc2U2NA=="`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &bytesOptions{}
	cmd := &cobra.Command{Use: "bytes"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)

	err := cmd.Flags().Set("raw", "flag-raw")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("hex", "666c61672d686578") // "flag-hex"
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("base64", "ZmxhZy1iYXNlNjQ=") // "flag-base64"
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []byte("flag-raw"), opts.Raw)
	assert.Equal(suite.T(), []byte("flag-hex"), []byte(opts.Hex))
	assert.Equal(suite.T(), []byte("flag-base64"), []byte(opts.Base64))
}

func (suite *structcliSuite) TestHooks_HexBytesInvalidFromYAML() {
	configContent := `hex: "invalid-hex"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &bytesOptions{}
	cmd := &cobra.Command{Use: "bytes"}

	loadConfigForCommand(suite.T(), cmd, configFile)

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid string for structcli.Hex")
	assert.Contains(suite.T(), err.Error(), "couldn't unmarshal config to options:")
}

type netOptions struct {
	Address net.IP     `flag:"address" flagdescr:"ip address" flagenv:"true"`
	Mask    net.IPMask `flag:"mask" flagdescr:"ip mask" flagenv:"true"`
	Network net.IPNet  `flag:"network" flagdescr:"ip network" flagenv:"true"`
	Peers   []net.IP   `flag:"peers" flagdescr:"peer addresses" flagenv:"true"`
}

func (o *netOptions) Attach(c *cobra.Command) error { return nil }

func assertIPSliceEqual(t *testing.T, expected []net.IP, actual []net.IP) {
	require.Len(t, actual, len(expected))

	for i := range expected {
		assert.True(t, expected[i].Equal(actual[i]), "IP at position %d should match", i)
	}
}

func assertIPNetEqual(t *testing.T, expected *net.IPNet, actual net.IPNet) {
	assert.True(t, expected.IP.Equal(actual.IP))
	assert.Equal(t, expected.Mask, actual.Mask)
}

func (suite *structcliSuite) TestHooks_NetTypesFromFlag() {
	opts := &netOptions{}
	cmd := &cobra.Command{Use: "net"}

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)

	err = cmd.Flags().Set("address", "10.42.0.10")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("mask", "ffffff00")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("network", "10.42.0.0/24")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("peers", "10.42.0.11,10.42.0.12")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("peers", "10.42.0.13")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)

	assert.True(suite.T(), net.ParseIP("10.42.0.10").Equal(opts.Address))
	assert.Equal(suite.T(), net.IPv4Mask(255, 255, 255, 0), opts.Mask)
	_, expectedNet, _ := net.ParseCIDR("10.42.0.0/24")
	assertIPNetEqual(suite.T(), expectedNet, opts.Network)
	assertIPSliceEqual(suite.T(), []net.IP{
		net.ParseIP("10.42.0.11"),
		net.ParseIP("10.42.0.12"),
		net.ParseIP("10.42.0.13"),
	}, opts.Peers)
}

func (suite *structcliSuite) TestHooks_NetTypesFromYAMLString() {
	configContent := `address: "192.168.10.10"
mask: "255.255.255.0"
network: "192.168.10.0/24"
peers: "192.168.10.11,192.168.10.12"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &netOptions{}
	cmd := &cobra.Command{Use: "net"}
	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)

	assert.True(suite.T(), net.ParseIP("192.168.10.10").Equal(opts.Address))
	assert.Equal(suite.T(), net.IPv4Mask(255, 255, 255, 0), opts.Mask)
	_, expectedNet, _ := net.ParseCIDR("192.168.10.0/24")
	assertIPNetEqual(suite.T(), expectedNet, opts.Network)
	assertIPSliceEqual(suite.T(), []net.IP{
		net.ParseIP("192.168.10.11"),
		net.ParseIP("192.168.10.12"),
	}, opts.Peers)
}

func (suite *structcliSuite) TestHooks_NetTypesFromYAMLArray() {
	configContent := `address: "172.16.10.10"
mask: "ffffff00"
network: "172.16.10.0/24"
peers:
  - "172.16.10.11"
  - "172.16.10.12"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &netOptions{}
	cmd := &cobra.Command{Use: "net"}
	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)

	assert.True(suite.T(), net.ParseIP("172.16.10.10").Equal(opts.Address))
	assert.Equal(suite.T(), net.IPv4Mask(255, 255, 255, 0), opts.Mask)
	_, expectedNet, _ := net.ParseCIDR("172.16.10.0/24")
	assertIPNetEqual(suite.T(), expectedNet, opts.Network)
	assertIPSliceEqual(suite.T(), []net.IP{
		net.ParseIP("172.16.10.11"),
		net.ParseIP("172.16.10.12"),
	}, opts.Peers)
}

func (suite *structcliSuite) TestHooks_NetTypesFromEnv() {
	const (
		envAddress = "NETAPP_ADDRESS"
		envMask    = "NETAPP_MASK"
		envNetwork = "NETAPP_NETWORK"
		envPeers   = "NETAPP_PEERS"
	)

	defer func() {
		structcli.SetEnvPrefix("")
	}()

	suite.T().Setenv(envAddress, "10.10.0.10")
	suite.T().Setenv(envMask, "ffffff00")
	suite.T().Setenv(envNetwork, "10.10.0.0/24")
	suite.T().Setenv(envPeers, "10.10.0.11,10.10.0.12")
	structcli.SetEnvPrefix("netapp")

	opts := &netOptions{}
	cmd := &cobra.Command{Use: "netapp"}

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)

	assert.True(suite.T(), net.ParseIP("10.10.0.10").Equal(opts.Address))
	assert.Equal(suite.T(), net.IPv4Mask(255, 255, 255, 0), opts.Mask)
	_, expectedNet, _ := net.ParseCIDR("10.10.0.0/24")
	assertIPNetEqual(suite.T(), expectedNet, opts.Network)
	assertIPSliceEqual(suite.T(), []net.IP{
		net.ParseIP("10.10.0.11"),
		net.ParseIP("10.10.0.12"),
	}, opts.Peers)
}

func (suite *structcliSuite) TestHooks_NetIPMaskInvalidFromYAML() {
	configContent := `address: "10.0.0.10"
mask: "not-a-mask"
network: "10.0.0.0/24"
peers: "10.0.0.11"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &netOptions{}
	cmd := &cobra.Command{Use: "net"}
	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid string for net.IPMask")
	assert.Contains(suite.T(), err.Error(), "couldn't unmarshal config to options:")
}

func (suite *structcliSuite) TestHooks_NetIPSliceInvalidFromYAML() {
	configContent := `address: "10.0.0.10"
mask: "ffffff00"
network: "10.0.0.0/24"
peers: "10.0.0.11,not-an-ip"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &netOptions{}
	cmd := &cobra.Command{Use: "net"}
	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)
	err = structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid string for []net.IP")
	assert.Contains(suite.T(), err.Error(), "not-an-ip")
	assert.Contains(suite.T(), err.Error(), "couldn't unmarshal config to options:")
}

type requiredWithEnvRuntimeOptions struct {
	RequiredEnvFlag string `flag:"required-env-flag" flagrequired:"true" flagenv:"true" flagdescr:"required flag with env"`
	OptionalEnvFlag string `flag:"optional-env-flag" flagenv:"true" flagdescr:"optional flag with env"`
}

func (o *requiredWithEnvRuntimeOptions) Attach(c *cobra.Command) error { return nil }

func (suite *structcliSuite) TestFlagrequired_WithEnvRuntimeBehavior() {
	suite.T().Run("required_flag_with_env_var_set", func(t *testing.T) {
		structcli.SetEnvPrefix("AUTOFLAGS")
		defer structcli.SetEnvPrefix("")

		// Set the environment variable that will be used
		envVarName := "AUTOFLAGS_TEST_REQUIRED_ENV_FLAG"
		suite.T().Setenv(envVarName, "env-value")

		opts := &requiredWithEnvRuntimeOptions{}
		cmd := &cobra.Command{Use: "test"}

		structcli.Define(cmd, opts)

		// Verify both annotations are set
		flags := cmd.Flags()
		requiredEnvFlag := flags.Lookup("required-env-flag")
		assert.NotNil(t, requiredEnvFlag, "required-env-flag should exist")

		// Should have both required and env annotations
		requiredAnnotation := requiredEnvFlag.Annotations[cobra.BashCompOneRequiredFlag]
		assert.NotNil(t, requiredAnnotation, "should have required annotation")
		assert.Equal(t, []string{"true"}, requiredAnnotation)

		envAnnotation := requiredEnvFlag.Annotations[internalenv.FlagAnnotation]
		assert.NotNil(t, envAnnotation, "should have env annotation")
		assert.Contains(t, envAnnotation, envVarName, "should contain the correct env var name")

		// Test that Unmarshal works with environment variable
		err := structcli.Unmarshal(cmd, opts)
		assert.NoError(t, err, "should unmarshal successfully with env var set")
		assert.Equal(t, "env-value", opts.RequiredEnvFlag, "should get value from environment")

		// Compare with optional env flag behavior
		optionalEnvFlag := flags.Lookup("optional-env-flag")
		assert.NotNil(t, optionalEnvFlag, "optional-env-flag should exist")

		optionalRequiredAnnotation := optionalEnvFlag.Annotations[cobra.BashCompOneRequiredFlag]
		assert.Nil(t, optionalRequiredAnnotation, "optional flag should not have required annotation")

		optionalEnvAnnotation := optionalEnvFlag.Annotations[internalenv.FlagAnnotation]
		assert.NotNil(t, optionalEnvAnnotation, "optional flag should have env annotation")
	})
}

func (suite *structcliSuite) TestFlagrequired_WithEnvMissingValue() {
	// Test what happens when a required+env flag has no env var set and no flag provided
	suite.T().Run("required_flag_with_no_env_var", func(t *testing.T) {
		structcli.SetEnvPrefix("AUTOFLAGS")
		defer structcli.SetEnvPrefix("")

		// Ensure the env vars are not set
		envVarNames := []string{
			"AUTOFLAGS_TEST_REQUIREDENVFLAG",
			"AUTOFLAGS_TEST_REQUIRED_ENV_FLAG",
		}

		for _, envVar := range envVarNames {
			unsetEnv(suite.T(), envVar)
		}

		opts := &requiredWithEnvRuntimeOptions{}
		cmd := &cobra.Command{Use: "test"}

		structcli.Define(cmd, opts)

		// Since the flag is required and no env var is set, this should work fine
		// because structcli doesn't enforce cobra's required flags during Unmarshal
		err := structcli.Unmarshal(cmd, opts)
		assert.NoError(t, err, "Unmarshal should succeed even with missing required flag")
		assert.Equal(t, "", opts.RequiredEnvFlag, "should have empty value when no env var or flag set")

		// The required enforcement would happen when cobra validates the command execution,
		// not during the structcli Unmarshal phase
	})
}

func (suite *structcliSuite) TestFlagrequired_WithEnvConfigFile() {
	// Test that required flags work with config files too
	suite.T().Run("required_flag_from_config", func(t *testing.T) {
		configContent := `required-env-flag: "config-value"`
		configFile := suite.createTempYAMLFile(configContent)
		defer os.Remove(configFile)

		opts := &requiredWithEnvRuntimeOptions{}
		cmd := &cobra.Command{Use: "test"}

		loadConfigForCommand(t, cmd, configFile)

		structcli.Define(cmd, opts)
		err := structcli.Unmarshal(cmd, opts)

		assert.NoError(t, err, "should unmarshal successfully with config file")
		assert.Equal(t, "config-value", opts.RequiredEnvFlag, "should get value from config")
	})
}

func (suite *structcliSuite) TestHooks_ZapcoreLevelFromYAML_InvalidLevel() {
	configContent := `loglevel: "invalidlevelstring"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &zapcoreLevelOptions{}
	cmd := &cobra.Command{Use: "testinvalidlevel"}

	structcli.Define(cmd, opts)

	loadConfigForCommand(suite.T(), cmd, configFile)

	err := structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err, "Unmarshal should return an error for invalid zapcore.Level")
	assert.Contains(suite.T(), err.Error(), "couldn't unmarshal config to options:", "Error should be wrapped by Unmarshal")
	assert.Contains(suite.T(), err.Error(), "invalidlevelstring", "Error should contain the invalid input value")
}

type slogLevelOptions struct {
	LogLevel slog.Level `default:"info" flag:"log-level" flagdescr:"the logging level" flagenv:"true"`
}

func (o *slogLevelOptions) Attach(c *cobra.Command) error { return nil }

func (suite *structcliSuite) TestHooks_DefineSlogLevelFlag() {
	// Test just defining the flag, without config file
	opts := &slogLevelOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := structcli.Define(cmd, opts)

	require.Nil(suite.T(), err)

	// Check if the flag was created
	flag := cmd.Flags().Lookup("log-level")
	assert.NotNil(suite.T(), flag)
	assert.Contains(suite.T(), flag.Usage, "{debug,info,warn,error}")
}

func (suite *structcliSuite) TestHooks_SlogLevelFromYAML() {
	// Create a temporary config file
	configContent := `log-level: debug`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	// Define options with slog.Level field
	opts := &slogLevelOptions{}

	cmd := &cobra.Command{Use: "test"}

	// Set up viper to read from our config file
	loadConfigForCommand(suite.T(), cmd, configFile)

	// Define flags and unmarshal
	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), slog.LevelDebug, opts.LogLevel)
}

func (suite *structcliSuite) TestHooks_SlogLevelFromEnv() {
	defer func() {
		// Reset state after test
		viper.Reset()
		structcli.SetEnvPrefix("")
	}()

	// Set environment variable (note: log-level becomes LOG_LEVEL)
	suite.T().Setenv("TESTAPP_LOG_LEVEL", "warn")

	// IMPORTANT: Set environment prefix BEFORE defining flags
	structcli.SetEnvPrefix("TESTAPP")

	opts := &slogLevelOptions{}
	cmd := &cobra.Command{Use: "testapp"}

	// Define flags and unmarshal
	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), slog.LevelWarn, opts.LogLevel)
}

func (suite *structcliSuite) TestHooks_SlogLevelFromFlag() {
	opts := &slogLevelOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Define flags
	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)

	// Set flag value directly instead of using Execute
	err = cmd.Flags().Set("log-level", "error")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), slog.LevelError, opts.LogLevel)
}

func (suite *structcliSuite) TestHooks_SlogLevelInvalidValue() {
	// Create a temporary config file with invalid level
	configContent := `log-level: invalid`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &slogLevelOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Set up viper to read from our config file
	loadConfigForCommand(suite.T(), cmd, configFile)

	// Define flags and unmarshal
	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid string for slog.Level")
}

func (suite *structcliSuite) TestHooks_SlogLevelWithOffsets() {
	// Test the ERROR+2, INFO-4 feature
	testCases := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR+2", slog.LevelError + 2},
		{"INFO-4", slog.LevelInfo - 4},
		{"WARN+1", slog.LevelWarn + 1},
		{"DEBUG-2", slog.LevelDebug - 2},
	}

	for _, tc := range testCases {
		suite.T().Run(fmt.Sprintf("level_%s", tc.input), func(t *testing.T) {
			// Reset viper state for each subtest
			viper.Reset()
			structcli.SetEnvPrefix("")

			// Create config file with the test level
			configContent := fmt.Sprintf(`log-level: %s`, tc.input)
			configFile := suite.createTempYAMLFile(configContent)
			defer os.Remove(configFile)

			opts := &slogLevelOptions{}
			cmd := &cobra.Command{Use: "test"}

			// Set up viper to read from our config file
			loadConfigForCommand(t, cmd, configFile)

			// Define flags and unmarshal
			err := structcli.Define(cmd, opts)
			require.NoError(t, err)

			err = structcli.Unmarshal(cmd, opts)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, opts.LogLevel,
				"Level %s should parse to %d but got %d", tc.input, tc.expected, opts.LogLevel)
		})
	}
}

func (suite *structcliSuite) TestStringToSlogLevelHookFunc_TypeGuard() {
	type testStruct struct {
		NotSlogLevel string `mapstructure:"level"`
	}

	opts := &testStruct{}

	// Use our slog hook directly with mapstructure
	// This will force the hook to be called even for non-slog.Level fields
	slogHook := internalhooks.StringToSlogLevelHookFunc()

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: slogHook, // Force slog hook to be used for all fields
		Result:     opts,
	})
	require.NoError(suite.T(), err)

	input := map[string]any{"level": "warn"}

	err = decoder.Decode(input)
	assert.NoError(suite.T(), err)

	// The string field should remain unchanged because the type guard
	// should have returned early when target type != slog.Level
	assert.Equal(suite.T(), "warn", opts.NotSlogLevel, "String should pass through unchanged due to type guard condition")
}

func (suite *structcliSuite) TestStringToZapcoreLevelHookFunc_TypeGuard() {
	type testStruct struct {
		NotZapcoreLevel string `mapstructure:"level"`
	}

	opts := &testStruct{}

	// Use the int enum hook for zapcore.Level directly with mapstructure
	// This will force the hook to be called even for non-zapcore.Level fields
	zapcoreHook := internalhooks.StringToIntEnumHookFunc(map[zapcore.Level][]string{
		zapcore.DebugLevel:  {"debug"},
		zapcore.InfoLevel:   {"info"},
		zapcore.WarnLevel:   {"warn"},
		zapcore.ErrorLevel:  {"error"},
		zapcore.DPanicLevel: {"dpanic"},
		zapcore.PanicLevel:  {"panic"},
		zapcore.FatalLevel:  {"fatal"},
	})

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: zapcoreHook, // Force zapcore hook to be used for all fields
		Result:     opts,
	})
	require.NoError(suite.T(), err)

	// Input data - string that would be valid for zapcore.Level
	input := map[string]any{"level": "error"}

	err = decoder.Decode(input)
	assert.NoError(suite.T(), err)

	// The string field should remain unchanged because the type guard
	// should have returned early when target type != zapcore.Level
	assert.Equal(suite.T(), "error", opts.NotZapcoreLevel, "String should pass through unchanged due to type guard condition")
}
