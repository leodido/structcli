package structcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	structclierrors "github.com/leodido/structcli/errors"
	"github.com/leodido/structcli/exitcode"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleError_NilError(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "test"}
	code := HandleError(cmd, nil, &buf)
	assert.Equal(t, exitcode.OK, code)
	assert.Empty(t, buf.String())
}

func TestHandleError_MissingRequiredFlag(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}
	srv := &cobra.Command{Use: "srv"}
	cmd.AddCommand(srv)

	err := fmt.Errorf(`required flag(s) "port" not set`)
	code := HandleError(srv, err, &buf)

	assert.Equal(t, exitcode.MissingRequiredFlag, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "missing_required_flag", se.Error)
	assert.Equal(t, exitcode.MissingRequiredFlag, se.ExitCode)
	assert.Equal(t, "port", se.Flag)
	assert.Equal(t, "mycli srv", se.Command)
	assert.Contains(t, se.Message, "port")
}

func TestHandleError_MissingRequiredFlagWithEnvHint(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}

	// Create a flag with env annotation
	cmd.Flags().IntP("port", "p", 0, "Server port")
	_ = cmd.MarkFlagRequired("port")
	_ = cmd.Flags().SetAnnotation("port", "___leodido_structcli_flagenvs", []string{"MYCLI_PORT"})

	err := fmt.Errorf(`required flag(s) "port" not set`)
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.MissingRequiredFlag, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "missing_required_flag", se.Error)
	assert.Equal(t, "port", se.Flag)
	assert.Contains(t, se.Hint, "MYCLI_PORT")
}

func TestHandleError_MissingRequiredMultipleFlags(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "test"}

	err := fmt.Errorf(`required flag(s) "port", "host" not set`)
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.MissingRequiredFlag, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "missing_required_flag", se.Error)
	// No single flag enrichment for multiple flags
	assert.Empty(t, se.Flag)
}

func TestHandleError_InvalidFlagValue(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}
	srv := &cobra.Command{Use: "srv"}
	cmd.AddCommand(srv)

	err := fmt.Errorf(`invalid argument "abc" for "-p, --port" flag: strconv.ParseInt: parsing "abc": invalid syntax`)
	code := HandleError(srv, err, &buf)

	assert.Equal(t, exitcode.InvalidFlagValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "invalid_flag_value", se.Error)
	assert.Equal(t, "port", se.Flag)
	assert.Equal(t, "abc", se.Got)
	assert.Equal(t, "mycli srv", se.Command)
}

func TestHandleError_InvalidFlagValueFromEnvVar(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}

	// Create a flag with env annotation
	cmd.Flags().IntP("port", "p", 0, "Server port")
	_ = cmd.Flags().SetAnnotation("port", "___leodido_structcli_flagenvs", []string{"MYCLI_PORT"})

	// Simulate: flag NOT changed on CLI, but env var IS set
	t.Setenv("MYCLI_PORT", "abc")

	err := fmt.Errorf(`invalid argument "abc" for "-p, --port" flag: strconv.ParseInt: parsing "abc": invalid syntax`)
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.EnvInvalidValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "env_invalid_value", se.Error)
	assert.Equal(t, "MYCLI_PORT", se.EnvVar)
	assert.Equal(t, "port", se.Flag)
	assert.Equal(t, "abc", se.Got)
}

func TestHandleError_UnknownFlag(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}

	err := fmt.Errorf("unknown flag: --nonexistent")
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.UnknownFlag, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "unknown_flag", se.Error)
	assert.Equal(t, "nonexistent", se.Flag)
}

func TestHandleError_UnknownCommand(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}
	srv := &cobra.Command{Use: "srv", Short: "Start server", RunE: func(c *cobra.Command, args []string) error { return nil }}
	usr := &cobra.Command{Use: "usr", Short: "User management", RunE: func(c *cobra.Command, args []string) error { return nil }}
	cmd.AddCommand(srv)
	cmd.AddCommand(usr)

	err := fmt.Errorf(`unknown command "bogus" for "mycli"`)
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.UnknownCommand, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "unknown_command", se.Error)
	assert.Equal(t, "bogus", se.Got)
	assert.Contains(t, se.Available, "srv")
	assert.Contains(t, se.Available, "usr")
}

func TestHandleError_ValidationFailed(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}

	err := &structclierrors.ValidationError{
		ContextName: "add",
		Errors: []error{
			fmt.Errorf("Field validation for 'Email' failed on the 'email' tag"),
			fmt.Errorf("Field validation for 'Age' failed on the 'min' tag"),
		},
	}
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.ValidationFailed, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "validation_failed", se.Error)
	assert.Len(t, se.Violations, 2)
	assert.Contains(t, se.Violations[0].Message, "Email")
	assert.Contains(t, se.Violations[1].Message, "Age")
}

func TestHandleError_InputError(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}

	err := structclierrors.NewInputError("string", "value too long")
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.InvalidFlagValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "invalid_input", se.Error)
}

func TestHandleError_ConfigUnknownKey(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}

	err := fmt.Errorf("unknown config keys: bogus, extra")
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.ConfigUnknownKey, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "config_unknown_key", se.Error)
	assert.Equal(t, "bogus", se.Key)
}

func TestHandleError_ConfigNotFound(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}

	err := fmt.Errorf("config file Not Found in [/etc/mycli]")
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.ConfigNotFound, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "config_not_found", se.Error)
}

func TestHandleError_ConfigParseError(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}

	err := fmt.Errorf("While parsing config: yaml: line 5: did not find expected key")
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.ConfigParseError, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "config_parse_error", se.Error)
}

func TestHandleError_GenericError(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}

	err := fmt.Errorf("connection refused")
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.Error, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "error", se.Error)
	assert.Equal(t, 1, se.ExitCode)
	assert.Equal(t, "connection refused", se.Message)
}

func TestHandleError_OutputIsValidJSON(t *testing.T) {
	cases := []error{
		fmt.Errorf(`required flag(s) "port" not set`),
		fmt.Errorf(`invalid argument "abc" for "--port" flag: bad`),
		fmt.Errorf("unknown flag: --foo"),
		fmt.Errorf(`unknown command "bar" for "test"`),
		fmt.Errorf("unknown config keys: baz"),
		fmt.Errorf("something unexpected"),
	}

	for _, err := range cases {
		var buf bytes.Buffer
		cmd := &cobra.Command{Use: "test"}
		HandleError(cmd, err, &buf)

		var raw json.RawMessage
		require.NoError(t, json.Unmarshal(buf.Bytes(), &raw), "output for %q should be valid JSON", err.Error())
	}
}

func TestHandleError_EnvMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}

	// Flag with env annotation, env var NOT set
	cmd.Flags().IntP("port", "p", 0, "Server port")
	_ = cmd.MarkFlagRequired("port")
	_ = cmd.Flags().SetAnnotation("port", "___leodido_structcli_flagenvs", []string{"MYCLI_PORT"})

	// Make sure env var is unset
	os.Unsetenv("MYCLI_PORT")

	err := fmt.Errorf(`required flag(s) "port" not set`)
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.MissingRequiredFlag, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "missing_required_flag", se.Error)
	assert.Equal(t, "port", se.Flag)
	assert.Contains(t, se.Hint, "MYCLI_PORT")
}

func TestHandleError_UnmarshalDecodeError_FromEnvVar(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "myapp"}
	cmd.Flags().IntP("port", "p", 0, "Server port")
	_ = cmd.Flags().SetAnnotation("port", "___leodido_structcli_flagenvs", []string{"MYAPP_PORT"})

	t.Setenv("MYAPP_PORT", "xyz")

	err := fmt.Errorf("couldn't unmarshal config to options: decoding failed due to the following error(s):\n\n'Port' cannot parse value as 'int': strconv.ParseInt: invalid syntax")
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.EnvInvalidValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "env_invalid_value", se.Error)
	assert.Equal(t, "MYAPP_PORT", se.EnvVar)
	assert.Equal(t, "port", se.Flag)
	assert.Equal(t, "xyz", se.Got)
	assert.Equal(t, "int", se.Expected)
}

func TestHandleError_UnmarshalDecodeError_NoEnvVar(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "myapp"}
	cmd.Flags().IntP("port", "p", 0, "Server port")

	err := fmt.Errorf("couldn't unmarshal config to options: decoding failed due to the following error(s):\n\n'Port' cannot parse value as 'int': strconv.ParseInt: invalid syntax")
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.InvalidFlagValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "invalid_flag_value", se.Error)
	assert.Equal(t, "port", se.Flag)
	assert.Equal(t, "int", se.Expected)
}

func TestHandleError_UnmarshalDecodeError_Unparseable(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "myapp"}

	err := fmt.Errorf("couldn't unmarshal config to options: decoding failed due to the following error(s):\n\nsome weird error")
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.InvalidFlagValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "invalid_flag_value", se.Error)
	assert.Empty(t, se.Flag)
}

// TestExtractLongFlagName tests the flag spec parser.
func TestExtractLongFlagName(t *testing.T) {
	tests := []struct {
		spec string
		want string
	}{
		{"-p, --port", "port"},
		{"--port", "port"},
		{"--log-level", "log-level"},
		{"-v, --verbose", "verbose"},
	}

	for _, tt := range tests {
		got := extractLongFlagName(tt.spec)
		assert.Equal(t, tt.want, got, "extractLongFlagName(%q)", tt.spec)
	}
}

// TestParseQuotedList tests cobra's quoted flag list parser.
func TestParseQuotedList(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{`"port"`, []string{"port"}},
		{`"port", "host"`, []string{"port", "host"}},
		{`"port", "host", "timeout"`, []string{"port", "host", "timeout"}},
	}

	for _, tt := range tests {
		got := parseQuotedList(tt.input)
		assert.Equal(t, tt.want, got, "parseQuotedList(%q)", tt.input)
	}
}

// TestHandleError_WrappedValidationError ensures errors.As works through wrapping.
func TestHandleError_WrappedValidationError(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "test"}

	inner := &structclierrors.ValidationError{
		ContextName: "user",
		Errors:      []error{fmt.Errorf("name is required")},
	}
	err := fmt.Errorf("unmarshal failed: %w", inner)
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.ValidationFailed, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "validation_failed", se.Error)
	assert.Len(t, se.Violations, 1)
}

type integrationRequiredOpts struct {
	Port int `flagshort:"p" flagdescr:"Server port" flagrequired:"true" flagenv:"true"`
}

func (o *integrationRequiredOpts) Attach(c *cobra.Command) error {
	return Define(c, o)
}

// Integration test: build a real structcli command and test HandleError with it.
func TestHandleError_Integration_RealCommand(t *testing.T) {
	o := &integrationRequiredOpts{}
	cmd := &cobra.Command{
		Use:           "myapp",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return Unmarshal(c, o)
		},
	}
	require.NoError(t, o.Attach(cmd))

	// Simulate execution with missing required flag
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)

	var buf bytes.Buffer
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.MissingRequiredFlag, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "missing_required_flag", se.Error)
	assert.Equal(t, "port", se.Flag)
	assert.Contains(t, se.Hint, "MYAPP_PORT")
}

type integrationValueOpts struct {
	Port int `flagshort:"p" flagdescr:"Server port"`
}

func (o *integrationValueOpts) Attach(c *cobra.Command) error {
	return Define(c, o)
}

// Integration test: invalid flag value through a real cobra execution.
func TestHandleError_Integration_InvalidValue(t *testing.T) {
	o := &integrationValueOpts{}
	cmd := &cobra.Command{
		Use:           "myapp",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return Unmarshal(c, o)
		},
	}
	require.NoError(t, o.Attach(cmd))

	cmd.SetArgs([]string{"--port", "abc"})
	err := cmd.Execute()
	require.Error(t, err)

	var buf bytes.Buffer
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.InvalidFlagValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "invalid_flag_value", se.Error)
	assert.Equal(t, "port", se.Flag)
	assert.Equal(t, "abc", se.Got)
}
