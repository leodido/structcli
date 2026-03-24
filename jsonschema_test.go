package structcli

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/leodido/structcli/jsonschema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

// jsonSchemaBasicOptions is a test fixture covering basic types, env vars, groups, and descriptions.
type jsonSchemaBasicOptions struct {
	Port     int          `flag:"port" flagshort:"p" flagdescr:"server port" flagenv:"true" flaggroup:"Network" flagrequired:"true"`
	Host     string       `flag:"host" flagdescr:"server host" default:"localhost" flagenv:"true" flaggroup:"Network"`
	LogLevel zapcore.Level `flag:"log-level" flagdescr:"set log level" flagenv:"true" flaggroup:"Logging"`
	Debug    bool         `flag:"debug" flagdescr:"enable debug mode"`
}

func (o *jsonSchemaBasicOptions) Attach(c *cobra.Command) error { return nil }

// jsonSchemaPresetOptions is a test fixture covering the preset annotation feature.
type jsonSchemaPresetOptions struct {
	Verbosity int `flag:"verbosity" flagdescr:"verbosity level" flagpreset:"verbose=5;quiet=0"`
}

func (o *jsonSchemaPresetOptions) Attach(c *cobra.Command) error { return nil }

// jsonSchemaNestedOptions is a test fixture covering nested struct support.
type jsonSchemaNestedOptions struct {
	Server jsonSchemaServerFlags
}

type jsonSchemaServerFlags struct {
	Addr string `flag:"addr" flagdescr:"listen address" flaggroup:"Server" default:"0.0.0.0"`
	Port int    `flag:"port" flagdescr:"listen port" flaggroup:"Server" default:"8080"`
}

func (o *jsonSchemaNestedOptions) Attach(c *cobra.Command) error { return nil }

// jsonSchemaNetOptions is a test fixture covering net.IP and related types.
type jsonSchemaNetOptions struct {
	IP   net.IP   `flag:"ip" flagdescr:"bind IP address" flagenv:"true"`
	CIDR net.IPNet `flag:"cidr" flagdescr:"network CIDR"`
}

func (o *jsonSchemaNetOptions) Attach(c *cobra.Command) error { return nil }

func TestJSONSchema_BasicFlags(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")
	SetEnvPrefix("myapp")

	cmd := &cobra.Command{Use: "serve", Short: "Start the server"}
	opts := &jsonSchemaBasicOptions{}
	require.NoError(t, Define(cmd, opts))

	schemas, err := JSONSchema(cmd)
	require.NoError(t, err)
	require.Len(t, schemas, 1)
	schema := schemas[0]

	assert.Equal(t, "serve", schema.Name)
	assert.Equal(t, "serve", schema.CommandPath)
	assert.Equal(t, "Start the server", schema.Description)
	assert.Equal(t, "MYAPP", schema.EnvPrefix)

	// Port flag
	portFlag, ok := schema.Flags["port"]
	require.True(t, ok, "port flag should exist")
	assert.Equal(t, "port", portFlag.Name)
	assert.Equal(t, "p", portFlag.Shorthand)
	assert.Equal(t, "int", portFlag.Type)
	assert.Equal(t, "server port", portFlag.Description)
	assert.True(t, portFlag.Required)
	assert.NotEmpty(t, portFlag.EnvVars, "port should have env vars")
	assert.Equal(t, "Network", portFlag.Group)

	// Host flag
	hostFlag, ok := schema.Flags["host"]
	require.True(t, ok, "host flag should exist")
	assert.Equal(t, "localhost", hostFlag.Default)
	assert.False(t, hostFlag.Required)
	assert.NotEmpty(t, hostFlag.EnvVars)

	// LogLevel flag - should have enum values extracted from usage
	logLevelFlag, ok := schema.Flags["log-level"]
	require.True(t, ok, "log-level flag should exist")
	assert.Equal(t, "Logging", logLevelFlag.Group)
	assert.NotEmpty(t, logLevelFlag.Enum, "log-level should have enum values")
	assert.Contains(t, logLevelFlag.Enum, "debug")
	assert.Contains(t, logLevelFlag.Enum, "info")
	assert.Contains(t, logLevelFlag.Enum, "error")

	// Groups
	assert.Contains(t, schema.Groups, "Network")
	assert.Contains(t, schema.Groups, "Logging")
	assert.Contains(t, schema.Groups["Network"], "port")
	assert.Contains(t, schema.Groups["Network"], "host")

	SetEnvPrefix("")
}

func TestJSONSchema_EnumInDescription(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	cmd := &cobra.Command{Use: "app"}
	opts := &jsonSchemaBasicOptions{}
	require.NoError(t, Define(cmd, opts))

	// Default: enum stripped from description
	schemas, err := JSONSchema(cmd)
	require.NoError(t, err)
	logFlag := schemas[0].Flags["log-level"]
	assert.Equal(t, "set log level", logFlag.Description)
	assert.NotEmpty(t, logFlag.Enum)

	// WithEnumInDescription: enum preserved in description
	schemas, err = JSONSchema(cmd, jsonschema.WithEnumInDescription())
	require.NoError(t, err)
	logFlag = schemas[0].Flags["log-level"]
	assert.Contains(t, logFlag.Description, "{")
	assert.NotEmpty(t, logFlag.Enum)
}

func TestJSONSchema_Presets(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	cmd := &cobra.Command{Use: "app"}
	opts := &jsonSchemaPresetOptions{}
	require.NoError(t, Define(cmd, opts))

	schemas, err := JSONSchema(cmd)
	require.NoError(t, err)
	schema := schemas[0]

	verbosityFlag, ok := schema.Flags["verbosity"]
	require.True(t, ok, "verbosity flag should exist")
	require.Len(t, verbosityFlag.Presets, 2)

	presetNames := map[string]string{}
	for _, p := range verbosityFlag.Presets {
		presetNames[p.Name] = p.Value
	}
	assert.Equal(t, "5", presetNames["verbose"])
	assert.Equal(t, "0", presetNames["quiet"])
}

func TestJSONSchema_NestedStructs(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	cmd := &cobra.Command{Use: "app"}
	opts := &jsonSchemaNestedOptions{}
	require.NoError(t, Define(cmd, opts))

	schemas, err := JSONSchema(cmd)
	require.NoError(t, err)
	schema := schemas[0]

	// Nested flags should be flattened
	addrFlag, ok := schema.Flags["addr"]
	require.True(t, ok, "addr flag should exist")
	assert.Equal(t, "0.0.0.0", addrFlag.Default)
	assert.Equal(t, "Server", addrFlag.Group)
	assert.Equal(t, "listen address", addrFlag.Description)

	portFlag, ok := schema.Flags["port"]
	require.True(t, ok, "port flag should exist")
	assert.Equal(t, "8080", portFlag.Default)
}

func TestJSONSchema_NetTypes(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")
	SetEnvPrefix("test")

	cmd := &cobra.Command{Use: "net"}
	opts := &jsonSchemaNetOptions{}
	require.NoError(t, Define(cmd, opts))

	schemas, err := JSONSchema(cmd)
	require.NoError(t, err)
	schema := schemas[0]

	ipFlag, ok := schema.Flags["ip"]
	require.True(t, ok, "ip flag should exist")
	assert.Equal(t, "bind IP address", ipFlag.Description)
	assert.NotEmpty(t, ipFlag.EnvVars)

	cidrFlag, ok := schema.Flags["cidr"]
	require.True(t, ok, "cidr flag should exist")
	assert.Equal(t, "network CIDR", cidrFlag.Description)

	SetEnvPrefix("")
}

func TestJSONSchema_WithFullTree(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	noop := func(c *cobra.Command, args []string) error { return nil }
	root := &cobra.Command{Use: "app", Short: "root command", RunE: noop}
	sub1 := &cobra.Command{Use: "serve", Short: "start server", RunE: noop}
	sub2 := &cobra.Command{Use: "config", Short: "manage config", RunE: noop}

	root.AddCommand(sub1, sub2)

	serveOpts := &jsonSchemaBasicOptions{}
	require.NoError(t, Define(sub1, serveOpts))

	schemas, err := JSONSchema(root, jsonschema.WithFullTree())
	require.NoError(t, err)
	require.Len(t, schemas, 3, "should have root + 2 subcommands")

	assert.Equal(t, "app", schemas[0].Name)
	assert.Contains(t, schemas[0].Subcommands, "serve")
	assert.Contains(t, schemas[0].Subcommands, "config")

	// Cobra sorts subcommands alphabetically: config before serve
	assert.Equal(t, "config", schemas[1].Name)
	assert.Equal(t, "app config", schemas[1].CommandPath)
	assert.Equal(t, "serve", schemas[2].Name)
	assert.Equal(t, "app serve", schemas[2].CommandPath)
}

func TestToJSONSchema_ValidOutput(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")
	SetEnvPrefix("myapp")

	cmd := &cobra.Command{Use: "serve", Short: "Start the server"}
	opts := &jsonSchemaBasicOptions{}
	require.NoError(t, Define(cmd, opts))

	schemas, err := JSONSchema(cmd)
	require.NoError(t, err)

	output, err := schemas[0].ToJSONSchema()
	require.NoError(t, err)

	// Verify it's valid JSON
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(output, &parsed))

	// Verify JSON Schema structure
	assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", parsed["$schema"])
	assert.Equal(t, "serve", parsed["title"])
	assert.Equal(t, "object", parsed["type"])

	// Verify properties exist
	props, ok := parsed["properties"].(map[string]any)
	require.True(t, ok, "properties should be an object")
	assert.Contains(t, props, "port")
	assert.Contains(t, props, "host")
	assert.Contains(t, props, "log-level")
	assert.Contains(t, props, "debug")

	// Verify port property
	portProp, ok := props["port"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "number", portProp["type"])
	assert.Equal(t, "server port", portProp["description"])
	assert.Equal(t, "p", portProp["x-structcli-shorthand"])
	assert.Equal(t, "Network", portProp["x-structcli-group"])
	assert.NotNil(t, portProp["x-structcli-env-vars"])

	// Verify host default
	hostProp := props["host"].(map[string]any)
	assert.Equal(t, "localhost", hostProp["default"])

	// Verify log-level enum
	logProp := props["log-level"].(map[string]any)
	assert.NotNil(t, logProp["enum"], "log-level should have enum")

	// Verify required
	required, ok := parsed["required"].([]any)
	require.True(t, ok, "required should be an array")
	assert.Contains(t, required, "port")

	// Verify extensions
	assert.Equal(t, "MYAPP", parsed["x-structcli-env-prefix"])

	SetEnvPrefix("")
}

func TestToJSONSchema_Presets(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	cmd := &cobra.Command{Use: "app"}
	opts := &jsonSchemaPresetOptions{}
	require.NoError(t, Define(cmd, opts))

	schemas, err := JSONSchema(cmd)
	require.NoError(t, err)

	output, err := schemas[0].ToJSONSchema()
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(output, &parsed))

	props := parsed["properties"].(map[string]any)
	verbProp := props["verbosity"].(map[string]any)

	presets, ok := verbProp["x-structcli-presets"].([]any)
	require.True(t, ok, "presets should be an array")
	assert.Len(t, presets, 2)
}

func TestToJSONSchema_TypeMapping(t *testing.T) {
	tests := []struct {
		pflagType    string
		expectedType string
		expectItems  bool
	}{
		{"bool", "boolean", false},
		{"int", "number", false},
		{"int64", "number", false},
		{"float64", "number", false},
		{"string", "string", false},
		{"duration", "string", false},
		{"stringSlice", "array", true},
		{"intSlice", "array", true},
		{"stringToString", "object", false},
	}

	for _, tc := range tests {
		t.Run(tc.pflagType, func(t *testing.T) {
			jsonType, items := pflagTypeToJSONSchemaType(tc.pflagType)
			assert.Equal(t, tc.expectedType, jsonType)
			if tc.expectItems {
				assert.NotNil(t, items)
			} else {
				assert.Nil(t, items)
			}
		})
	}
}
