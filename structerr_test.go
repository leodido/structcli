package structcli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/go-playground/validator/v10"
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

func TestHandleError_MissingRequiredFlagDoesNotClassifyEnvAsMissing(t *testing.T) {
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
	assert.Equal(t, exitcode.MissingRequiredFlag, se.ExitCode)
	assert.Equal(t, "port", se.Flag)
	assert.Empty(t, se.EnvVar)
	assert.Empty(t, se.Hint)
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

func TestHandleError_MissingRequiredFlagIgnoresUnsetEnvBindings(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}

	// Flag with multiple env annotations, all unset
	cmd.Flags().IntP("port", "p", 0, "Server port")
	_ = cmd.MarkFlagRequired("port")
	_ = cmd.Flags().SetAnnotation("port", "___leodido_structcli_flagenvs", []string{"MYCLI_PORT", "MYCLI_PORT_ALT"})

	// Make sure env var is unset
	os.Unsetenv("MYCLI_PORT")
	os.Unsetenv("MYCLI_PORT_ALT")

	err := fmt.Errorf(`required flag(s) "port" not set`)
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.MissingRequiredFlag, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "missing_required_flag", se.Error)
	assert.Equal(t, exitcode.MissingRequiredFlag, se.ExitCode)
	assert.Equal(t, "port", se.Flag)
	assert.Empty(t, se.EnvVar)
	assert.Empty(t, se.Hint)
}

// When an env var IS set but cobra still fires "required flag not set",
// it's because cobra checks required flags before viper merges env vars.
// The error remains about the missing required flag, not the env binding.
func TestHandleError_MissingRequiredFlagWithEnvVarSet(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}

	cmd.Flags().IntP("port", "p", 0, "Server port")
	_ = cmd.MarkFlagRequired("port")
	_ = cmd.Flags().SetAnnotation("port", "___leodido_structcli_flagenvs", []string{"MYCLI_PORT"})

	// Env var IS set with a valid value — cobra still complains because it doesn't check env vars
	t.Setenv("MYCLI_PORT", "3000")

	err := fmt.Errorf(`required flag(s) "port" not set`)
	code := HandleError(cmd, err, &buf)

	// Should still be missing_required_flag, NOT env_invalid_value
	assert.Equal(t, exitcode.MissingRequiredFlag, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "missing_required_flag", se.Error)
	assert.Equal(t, "port", se.Flag)
	assert.Empty(t, se.EnvVar)
	assert.Empty(t, se.Hint)
	// NOT env_invalid_value — the env var value is fine
	assert.NotEqual(t, "env_invalid_value", se.Error)
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

	assert.Equal(t, exitcode.ConfigInvalidValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "config_invalid_value", se.Error)
	assert.Equal(t, exitcode.ConfigInvalidValue, se.ExitCode)
	assert.Equal(t, "port", se.Flag)
	assert.Equal(t, "int", se.Expected)
}

func TestHandleError_UnmarshalDecodeError_Unparseable(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "myapp"}

	err := fmt.Errorf("couldn't unmarshal config to options: decoding failed due to the following error(s):\n\nsome weird error")
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.ConfigInvalidValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "config_invalid_value", se.Error)
	assert.Equal(t, exitcode.ConfigInvalidValue, se.ExitCode)
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

// TestHandleError_UnmarshalDecodeError_FieldPathAnnotation tests that findFlagForField
// matches via the field path annotation when the Go field name (LogLevel) differs from
// the flag name (level).
func TestHandleError_UnmarshalDecodeError_FieldPathAnnotation(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "myapp"}
	cmd.Flags().String("level", "info", "log level")
	// Simulate structcli's field path annotation: flag "level" has path "loglevel"
	_ = cmd.Flags().SetAnnotation("level", "___leodido_structcli_flagpath", []string{"loglevel"})
	_ = cmd.Flags().SetAnnotation("level", "___leodido_structcli_flagenvs", []string{"MYAPP_LEVEL"})

	t.Setenv("MYAPP_LEVEL", "bogus")

	// mapstructure error uses Go field name "LogLevel", not flag name "level"
	err := fmt.Errorf("couldn't unmarshal config to options: decoding failed due to the following error(s):\n\n'LogLevel' invalid string for zapcore.Level 'bogus': unrecognized level: \"bogus\"")
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.EnvInvalidValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "env_invalid_value", se.Error)
	assert.Equal(t, "level", se.Flag) // Resolved to flag name, not Go field name
	assert.Equal(t, "MYAPP_LEVEL", se.EnvVar)
	assert.Equal(t, "bogus", se.Got)
}

// TestParseDecodeError tests the three regex patterns for mapstructure errors.
func TestParseDecodeError(t *testing.T) {
	tests := []struct {
		name         string
		errMsg       string
		wantField    string
		wantGot      string
		wantExpected string
	}{
		{
			name:         "pattern1_parse_value",
			errMsg:       "'Port' cannot parse value as 'int': strconv.ParseInt: invalid syntax",
			wantField:    "Port",
			wantGot:      "",
			wantExpected: "int",
		},
		{
			name:         "pattern2_invalid_string",
			errMsg:       "'LogLevel' invalid string for zapcore.Level 'bogus': unrecognized level: \"bogus\"",
			wantField:    "LogLevel",
			wantGot:      "bogus",
			wantExpected: "",
		},
		{
			name:         "pattern2_invalid_value",
			errMsg:       "'MyField' invalid value for custom.Type 'bad': some error",
			wantField:    "MyField",
			wantGot:      "bad",
			wantExpected: "",
		},
		{
			name:         "pattern3_field_name_only",
			errMsg:       "'Timeout' some completely unknown error format",
			wantField:    "Timeout",
			wantGot:      "",
			wantExpected: "",
		},
		{
			name:      "no_match",
			errMsg:    "completely unparseable error with no quoted field",
			wantField: "",
			wantGot:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, got, expected := parseDecodeError(tt.errMsg)
			assert.Equal(t, tt.wantField, field)
			assert.Equal(t, tt.wantGot, got)
			assert.Equal(t, tt.wantExpected, expected)
		})
	}
}

// TestHandleError_InvalidFlagValueFromEnvVar_CobraPath tests source attribution
// when cobra itself catches an invalid flag value set from an env var.
func TestHandleError_InvalidFlagValueFromEnvVar_CobraPath(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "myapp"}
	cmd.Flags().IntP("port", "p", 0, "Server port")
	_ = cmd.Flags().SetAnnotation("port", "___leodido_structcli_flagenvs", []string{"MYAPP_PORT"})

	// Flag NOT changed on CLI, env var IS set
	t.Setenv("MYAPP_PORT", "not_a_number")

	err := fmt.Errorf(`invalid argument "not_a_number" for "-p, --port" flag: strconv.ParseInt: parsing "not_a_number": invalid syntax`)
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.EnvInvalidValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "env_invalid_value", se.Error)
	assert.Equal(t, "MYAPP_PORT", se.EnvVar)
	assert.Equal(t, "not_a_number", se.Got)
	assert.Equal(t, "int", se.Expected)
	assert.Equal(t, "port", se.Flag)
}

// TestExtractLongFlagName_Fallback tests the fallback path with no -- prefix.
func TestExtractLongFlagName_Fallback(t *testing.T) {
	// Edge case: spec with only short flag (shouldn't happen in cobra, but test the fallback)
	got := extractLongFlagName("-p")
	assert.Equal(t, "p", got)
}

// TestFindFlagForField tests both direct name match and path annotation match.
func TestFindFlagForField(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("level", "", "log level")
	_ = cmd.Flags().SetAnnotation("level", "___leodido_structcli_flagpath", []string{"loglevel"})
	cmd.Flags().Int("port", 0, "port")

	// Direct match
	assert.Equal(t, "port", findFlagForField(cmd, "Port"))
	assert.Equal(t, "port", findFlagForField(cmd, "port"))

	// Path annotation match
	assert.Equal(t, "level", findFlagForField(cmd, "LogLevel"))
	assert.Equal(t, "level", findFlagForField(cmd, "loglevel"))

	// No match
	assert.Equal(t, "", findFlagForField(cmd, "NonExistent"))
}

// TestFlagType tests the flagType helper.
func TestFlagType(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Int("port", 0, "port")
	cmd.Flags().String("host", "", "host")

	assert.Equal(t, "int", flagType(cmd, "port"))
	assert.Equal(t, "string", flagType(cmd, "host"))
	assert.Equal(t, "", flagType(cmd, "nonexistent"))
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
	assert.Equal(t, exitcode.MissingRequiredFlag, se.ExitCode)
	assert.Equal(t, "port", se.Flag)
	assert.Empty(t, se.EnvVar)
	assert.Empty(t, se.Hint)
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

// --- Validation Details tests ---

type validationDetailsTestStruct struct {
	Email string `validate:"required,email"`
	Age   int    `validate:"min=18"`
}

// TestHandleError_ValidationFailed_WithDetails tests that real validator errors
// produce Violations with Field (as flag name), Rule, Param, and Value populated.
func TestHandleError_ValidationFailed_WithDetails(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}
	cmd.Flags().String("email", "", "user email")
	cmd.Flags().Int("age", 0, "user age")

	// Create real validation errors via go-playground/validator
	v := validator.New()
	err := v.Struct(&validationDetailsTestStruct{
		Email: "not-an-email",
		Age:   10,
	})
	require.Error(t, err)

	var valErrs validator.ValidationErrors
	require.ErrorAs(t, err, &valErrs)

	// Wrap them in a structcli ValidationError
	errs := make([]error, len(valErrs))
	for i, fe := range valErrs {
		errs[i] = fe
	}
	ve := &structclierrors.ValidationError{
		ContextName: "test",
		Errors:      errs,
	}
	code := HandleError(cmd, ve, &buf)
	assert.Equal(t, exitcode.ValidationFailed, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "validation_failed", se.Error)
	require.Len(t, se.Violations, 2)

	// Find violations by field name
	violationsByField := map[string]Violation{}
	for _, v := range se.Violations {
		violationsByField[v.Field] = v
	}

	// Email violation: field resolved to flag name "email", rule "email"
	emailV, ok := violationsByField["email"]
	require.True(t, ok, "should have violation for field 'email', got fields: %v", keys(violationsByField))
	assert.Equal(t, "email", emailV.Rule)
	assert.NotEmpty(t, emailV.Message)

	// Age violation: field resolved to flag name "age", rule "min", param "18"
	ageV, ok := violationsByField["age"]
	require.True(t, ok, "should have violation for field 'age', got fields: %v", keys(violationsByField))
	assert.Equal(t, "min", ageV.Rule)
	assert.Equal(t, "18", ageV.Param)
	assert.NotEmpty(t, ageV.Message)
}

// TestHandleError_ValidationFailed_FieldToFlagMapping tests that
// ValidationDetail.StructField "LogLevel" maps to flag "level" via path annotation.
func TestHandleError_ValidationFailed_FieldToFlagMapping(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}
	cmd.Flags().String("level", "info", "log level")
	// structcli stores lowercase field path in annotation
	_ = cmd.Flags().SetAnnotation("level", flagPathAnnotation, []string{"loglevel"})

	// Create a validation error with StructField="LogLevel" using real validator
	type logOptions struct {
		LogLevel string `validate:"required"`
	}
	v := validator.New()
	err := v.Struct(&logOptions{LogLevel: ""})
	require.Error(t, err)

	var valErrs validator.ValidationErrors
	require.ErrorAs(t, err, &valErrs)

	errs := make([]error, len(valErrs))
	for i, fe := range valErrs {
		errs[i] = fe
	}
	ve := &structclierrors.ValidationError{
		ContextName: "log",
		Errors:      errs,
	}
	code := HandleError(cmd, ve, &buf)
	assert.Equal(t, exitcode.ValidationFailed, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	require.Len(t, se.Violations, 1)
	// StructField "LogLevel" should resolve to flag name "level" via path annotation
	assert.Equal(t, "level", se.Violations[0].Field)
	assert.Equal(t, "required", se.Violations[0].Rule)
}

// --- Enum violation tests ---

// TestHandleError_InvalidFlagEnum tests that a flag with enum annotation and a bad value
// produces exit code 15 (InvalidFlagEnum) with the available values array.
func TestHandleError_InvalidFlagEnum(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}
	cmd.Flags().String("mode", "fast", "Set mode {fast,slow,balanced}")
	_ = cmd.Flags().SetAnnotation("mode", flagEnumAnnotation, []string{"fast", "slow", "balanced"})

	// Cobra error for invalid argument
	err := fmt.Errorf(`invalid argument "turbo" for "--mode" flag: invalid value`)
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.InvalidFlagEnum, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "invalid_flag_enum", se.Error)
	assert.Equal(t, exitcode.InvalidFlagEnum, se.ExitCode)
	assert.Equal(t, "mode", se.Flag)
	assert.Equal(t, "turbo", se.Got)
	assert.Equal(t, "fast, slow, balanced", se.Expected)
	assert.Equal(t, []string{"fast", "slow", "balanced"}, se.Available)
	assert.Equal(t, "mycli", se.Command)
}

// TestHandleError_InvalidFlagEnum_ValidValue tests that when a flag has enum annotation
// but the value IS valid (and the error is a type error), it still goes through as invalid_flag_value.
func TestHandleError_InvalidFlagEnum_ValidValue(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}
	cmd.Flags().Int("priority", 1, "Priority level {1,2,3}")
	_ = cmd.Flags().SetAnnotation("priority", flagEnumAnnotation, []string{"1", "2", "3"})

	// Cobra error for invalid argument where "abc" is NOT in enum set
	// This tests: value not in enum AND type error — enum takes precedence
	err := fmt.Errorf(`invalid argument "abc" for "--priority" flag: strconv.ParseInt: parsing "abc": invalid syntax`)
	code := HandleError(cmd, err, &buf)

	// "abc" is not in the enum set, so it should be invalid_flag_enum
	assert.Equal(t, exitcode.InvalidFlagEnum, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "invalid_flag_enum", se.Error)
}

// TestHandleError_InvalidFlagEnum_UnmarshalPath tests enum detection through the unmarshal error path.
func TestHandleError_InvalidFlagEnum_UnmarshalPath(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "myapp"}
	cmd.Flags().String("level", "info", "log level")
	_ = cmd.Flags().SetAnnotation("level", flagPathAnnotation, []string{"loglevel"})
	_ = cmd.Flags().SetAnnotation("level", flagEnumAnnotation, []string{"debug", "info", "warn", "error"})

	// mapstructure decode error with invalid value for enum field
	err := fmt.Errorf("couldn't unmarshal config to options: decoding failed due to the following error(s):\n\n'LogLevel' invalid string for zapcore.Level 'bogus': unrecognized level: \"bogus\"")
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.InvalidFlagEnum, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "invalid_flag_enum", se.Error)
	assert.Equal(t, "level", se.Flag)
	assert.Equal(t, "bogus", se.Got)
	assert.Contains(t, se.Available, "debug")
	assert.Contains(t, se.Available, "info")
	assert.Contains(t, se.Available, "warn")
	assert.Contains(t, se.Available, "error")
}

// TestHandleError_MissingRequiredFlagWithValidateHint tests that when a flag has
// a validation annotation with "required", the hint includes this information.
func TestHandleError_MissingRequiredFlagWithValidateHint(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "mycli"}
	cmd.Flags().String("email", "", "user email")
	_ = cmd.MarkFlagRequired("email")
	_ = cmd.Flags().SetAnnotation("email", "___leodido_structcli_flagenvs", []string{"MYCLI_EMAIL"})
	_ = cmd.Flags().SetAnnotation("email", flagValidateAnnotation, []string{"required,email"})

	os.Unsetenv("MYCLI_EMAIL")

	err := fmt.Errorf(`required flag(s) "email" not set`)
	code := HandleError(cmd, err, &buf)

	assert.Equal(t, exitcode.MissingRequiredFlag, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "missing_required_flag", se.Error)
	assert.Equal(t, exitcode.MissingRequiredFlag, se.ExitCode)
	assert.Equal(t, "email", se.Flag)
	assert.Empty(t, se.EnvVar)
	assert.Contains(t, se.Hint, "required by validation")
}

// TestFlagEnumValues tests the flagEnumValues helper.
func TestFlagEnumValues(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("mode", "", "mode")
	_ = cmd.Flags().SetAnnotation("mode", flagEnumAnnotation, []string{"fast", "slow"})
	cmd.Flags().String("name", "", "name")

	assert.Equal(t, []string{"fast", "slow"}, flagEnumValues(cmd, "mode"))
	assert.Nil(t, flagEnumValues(cmd, "name"))
	assert.Nil(t, flagEnumValues(cmd, "nonexistent"))
}

// TestFlagValidateRules tests the flagValidateRules helper.
func TestFlagValidateRules(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("email", "", "email")
	_ = cmd.Flags().SetAnnotation("email", flagValidateAnnotation, []string{"required,email"})
	cmd.Flags().String("plain", "", "plain")

	assert.Equal(t, "required,email", flagValidateRules(cmd, "email"))
	assert.Equal(t, "", flagValidateRules(cmd, "plain"))
	assert.Equal(t, "", flagValidateRules(cmd, "nonexistent"))
}

// TestContains tests the contains helper.
func TestContains(t *testing.T) {
	assert.True(t, contains([]string{"a", "b", "c"}, "b"))
	assert.False(t, contains([]string{"a", "b", "c"}, "d"))
	assert.False(t, contains(nil, "a"))
	assert.False(t, contains([]string{}, "a"))
}

// keys is a test helper that returns map keys.
func keys(m map[string]Violation) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}

	return result
}

// --- Tests for SetupFlagErrors (typed FlagError path, no regex at classification time) ---

func TestSetupFlagErrors_InvalidValue(t *testing.T) {
	cmd := &cobra.Command{Use: "test", RunE: func(c *cobra.Command, args []string) error { return nil }}
	cmd.Flags().IntP("port", "p", 3000, "Server port")
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	SetupFlagErrors(cmd)

	cmd.SetArgs([]string{"--port", "abc"})
	err := cmd.Execute()
	require.Error(t, err)

	// Verify the error is a typed FlagError
	var flagErr *structclierrors.FlagError
	require.True(t, errors.As(err, &flagErr), "error should be a FlagError")
	assert.Equal(t, structclierrors.FlagErrorInvalidValue, flagErr.Kind)
	assert.Equal(t, "port", flagErr.FlagName)
	assert.Equal(t, "abc", flagErr.Value)

	// Verify HandleError classifies it correctly via errors.As (not regex)
	var buf bytes.Buffer
	code := HandleError(cmd, err, &buf)
	assert.Equal(t, exitcode.InvalidFlagValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "invalid_flag_value", se.Error)
	assert.Equal(t, "port", se.Flag)
	assert.Equal(t, "abc", se.Got)
	assert.Equal(t, "int", se.Expected)
}

func TestSetupFlagErrors_UnknownFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test", RunE: func(c *cobra.Command, args []string) error { return nil }}
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	SetupFlagErrors(cmd)

	cmd.SetArgs([]string{"--nonexistent"})
	err := cmd.Execute()
	require.Error(t, err)

	// Verify the error is a typed FlagError
	var flagErr *structclierrors.FlagError
	require.True(t, errors.As(err, &flagErr), "error should be a FlagError")
	assert.Equal(t, structclierrors.FlagErrorUnknown, flagErr.Kind)
	assert.Equal(t, "nonexistent", flagErr.FlagName)

	// Verify HandleError classifies it correctly
	var buf bytes.Buffer
	code := HandleError(cmd, err, &buf)
	assert.Equal(t, exitcode.UnknownFlag, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "unknown_flag", se.Error)
	assert.Equal(t, "nonexistent", se.Flag)
}

func TestSetupFlagErrors_InvalidValueWithShortFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test", RunE: func(c *cobra.Command, args []string) error { return nil }}
	cmd.Flags().IntP("port", "p", 3000, "Server port")
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	SetupFlagErrors(cmd)

	cmd.SetArgs([]string{"-p", "xyz"})
	err := cmd.Execute()
	require.Error(t, err)

	var flagErr *structclierrors.FlagError
	require.True(t, errors.As(err, &flagErr))
	assert.Equal(t, "port", flagErr.FlagName, "should extract long flag name even from short flag usage")
	assert.Equal(t, "xyz", flagErr.Value)
}

func TestSetupFlagErrors_EnumViolation(t *testing.T) {
	cmd := &cobra.Command{Use: "test", RunE: func(c *cobra.Command, args []string) error { return nil }}
	cmd.Flags().String("mode", "fast", "Set mode {fast,slow}")
	// Simulate enum annotation (normally set by Define)
	cmd.Flags().SetAnnotation("mode", flagEnumAnnotation, []string{"fast", "slow"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	SetupFlagErrors(cmd)

	// Create a FlagError for an invalid enum value
	flagErr := structclierrors.NewFlagError(structclierrors.FlagErrorInvalidValue, "mode", "turbo", fmt.Errorf("invalid argument"))

	var buf bytes.Buffer
	code := HandleError(cmd, flagErr, &buf)
	assert.Equal(t, exitcode.InvalidFlagEnum, code, "enum violation should use InvalidFlagEnum exit code")

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "invalid_flag_enum", se.Error)
	assert.Equal(t, "mode", se.Flag)
	assert.Equal(t, "turbo", se.Got)
	assert.Equal(t, []string{"fast", "slow"}, se.Available)
}

func TestSetupFlagErrors_FallbackWithoutSetup(t *testing.T) {
	// When SetupFlagErrors is NOT called, regex fallback still works
	cmd := &cobra.Command{Use: "test", RunE: func(c *cobra.Command, args []string) error { return nil }}
	cmd.Flags().IntP("port", "p", 3000, "Server port")
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	// Intentionally NOT calling SetupFlagErrors

	cmd.SetArgs([]string{"--port", "abc"})
	err := cmd.Execute()
	require.Error(t, err)

	// Error should NOT be a FlagError
	var flagErr *structclierrors.FlagError
	assert.False(t, errors.As(err, &flagErr), "without SetupFlagErrors, error should not be typed")

	// But HandleError should still classify it via regex
	var buf bytes.Buffer
	code := HandleError(cmd, err, &buf)
	assert.Equal(t, exitcode.InvalidFlagValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "invalid_flag_value", se.Error)
	assert.Equal(t, "port", se.Flag)
	assert.Equal(t, "abc", se.Got)
}

// --- Coverage gap tests ---

func TestSetupFlagErrors_EnvVarSourceAttribution(t *testing.T) {
	cmd := &cobra.Command{Use: "test", RunE: func(c *cobra.Command, args []string) error { return nil }}
	cmd.Flags().IntP("port", "p", 3000, "Server port")
	cmd.Flags().SetAnnotation("port", "___leodido_structcli_flagenvs", []string{"TEST_PORT"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	flagErr := structclierrors.NewFlagError(structclierrors.FlagErrorInvalidValue, "port", "abc", fmt.Errorf("invalid argument"))

	t.Setenv("TEST_PORT", "abc")

	var buf bytes.Buffer
	code := HandleError(cmd, flagErr, &buf)
	assert.Equal(t, exitcode.EnvInvalidValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "env_invalid_value", se.Error)
	assert.Equal(t, "TEST_PORT", se.EnvVar)
	assert.Equal(t, "port", se.Flag)
	assert.Equal(t, "abc", se.Got)
	assert.Equal(t, "int", se.Expected)
}

func TestHandleError_ValidationFailed_EmptyDetailsWithErrors(t *testing.T) {
	ve := &structclierrors.ValidationError{
		ContextName: "test",
		Errors:      []error{fmt.Errorf("custom validation: field X is bad"), nil, fmt.Errorf("another problem")},
	}

	cmd := &cobra.Command{Use: "test"}
	var buf bytes.Buffer
	code := HandleError(cmd, ve, &buf)
	assert.Equal(t, exitcode.ValidationFailed, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "validation_failed", se.Error)
	assert.Len(t, se.Violations, 2)
	assert.Equal(t, "custom validation: field X is bad", se.Violations[0].Message)
	assert.Equal(t, "another problem", se.Violations[1].Message)
}

func TestHandleError_InvalidFlagValue_FromEnvVarRegexFallback(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().IntP("port", "p", 3000, "Server port")
	cmd.Flags().SetAnnotation("port", "___leodido_structcli_flagenvs", []string{"TEST_PORT"})

	t.Setenv("TEST_PORT", "abc")

	err := fmt.Errorf(`invalid argument "abc" for "-p, --port" flag: strconv.ParseInt: parsing "abc": invalid syntax`)

	var buf bytes.Buffer
	code := HandleError(cmd, err, &buf)
	assert.Equal(t, exitcode.EnvInvalidValue, code)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "env_invalid_value", se.Error)
	assert.Equal(t, "TEST_PORT", se.EnvVar)
}

func TestHandleError_FlagError_UsesCommandPath(t *testing.T) {
	// FlagError no longer carries CommandPath — HandleError uses cmd.CommandPath()
	root := &cobra.Command{Use: "myapp"}
	srv := &cobra.Command{Use: "srv", RunE: func(c *cobra.Command, args []string) error { return nil }}
	root.AddCommand(srv)

	fe := structclierrors.NewFlagError(structclierrors.FlagErrorInvalidValue, "port", "abc", fmt.Errorf("bad"))

	var buf bytes.Buffer
	// Pass the subcommand (as ExecuteC would)
	HandleError(srv, fe, &buf)

	var se StructuredError
	require.NoError(t, json.Unmarshal(buf.Bytes(), &se))
	assert.Equal(t, "myapp srv", se.Command, "should use cmd.CommandPath() from the subcommand")
}

func TestHandleError_FlagError_ValidEnumValue(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	fe := structclierrors.NewFlagError(structclierrors.FlagErrorInvalidValue, "mode", "fast", fmt.Errorf("bad"))

	var buf bytes.Buffer
	code := HandleError(cmd, fe, &buf)
	assert.Equal(t, exitcode.InvalidFlagValue, code, "valid enum value should not be classified as enum violation")
}
