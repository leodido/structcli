package structcli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	structclierrors "github.com/leodido/structcli/errors"
	"github.com/leodido/structcli/exitcode"
	internalenv "github.com/leodido/structcli/internal/env"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Regex patterns for matching cobra's untyped error strings.
// These are ONLY used as fallbacks when SetupFlagErrors has not been called
// (ie. the CLI author did not opt into typed flag error interception).
// When SetupFlagErrors is active, flag errors are intercepted as typed FlagError
// via errors.As — no regex needed for those paths.
var (
	// "required flag(s) "port" not set" or "required flag(s) "port", "host" not set"
	reRequiredFlags = regexp.MustCompile(`^required flag\(s\) (.+) not set$`)
	// `invalid argument "abc" for "-p, --port" flag: ...`
	reInvalidArg = regexp.MustCompile(`^invalid argument "([^"]*)" for "([^"]*)" flag`)
	// `unknown flag: --foo`
	reUnknownFlag = regexp.MustCompile(`^unknown flag: --(.+)$`)
	// `unknown command "foo" for "mycli"`
	reUnknownCommand = regexp.MustCompile(`^unknown command "([^"]*)" for "([^"]*)"`)
	// `unknown config keys: extra, bogus`
	reConfigUnknownKeys = regexp.MustCompile(`^unknown config keys: (.+)$`)
	// `'Port' cannot parse value as 'int': strconv.ParseInt: invalid syntax`
	reDecodeFieldError = regexp.MustCompile(`'(\w+)' cannot parse value as '([\w.]+)'`)
	// `'LogLevel' invalid string for zapcore.Level 'bogus': unrecognized level: "bogus"`
	reDecodeFieldInvalid = regexp.MustCompile(`'(\w+)' (?:invalid (?:string|value) for [\w.]+ )'([^']*)'`)
	// General: extract just the field name from `'FieldName' ...`
	reDecodeFieldName = regexp.MustCompile(`'(\w+)'`)

	// Used by SetupFlagErrors to parse pflag error strings at interception time.
	// These patterns match the exact formats pflag produces in flag.go:
	//   fmt.Errorf("invalid argument %q for %q flag: %v", value, flagName, err)
	//   f.failf("unknown flag: --%s", name)
	rePflagInvalidArg  = regexp.MustCompile(`^invalid argument "([^"]*)" for "([^"]*)" flag`)
	rePflagUnknownFlag = regexp.MustCompile(`^unknown flag: --(.+)$`)
)

// SetupFlagErrors installs a flag error interceptor on the root command.
//
// When active, cobra's flag parsing errors (invalid values, unknown flags) are
// wrapped in typed [structclierrors.FlagError] values. [HandleError] then uses
// errors.As to classify them — no regex parsing at classification time.
//
// Call this on the root command before Execute():
//
//	structcli.SetupFlagErrors(rootCmd)
//
// This is optional. If not called, HandleError falls back to regex-based
// classification of cobra's string errors.
func SetupFlagErrors(rootC *cobra.Command) {
	rootC.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		if err == nil {
			return nil
		}
		errMsg := err.Error()

		// Parse pflag's "invalid argument" format:
		//   invalid argument "abc" for "-p, --port" flag: strconv.ParseInt: ...
		if m := rePflagInvalidArg.FindStringSubmatch(errMsg); m != nil {
			return structclierrors.NewFlagError(structclierrors.FlagErrorInvalidValue, extractLongFlagName(m[2]), m[1], err)
		}

		// Parse pflag's "unknown flag" format:
		//   unknown flag: --foo
		if m := rePflagUnknownFlag.FindStringSubmatch(errMsg); m != nil {
			return structclierrors.NewFlagError(structclierrors.FlagErrorUnknown, m[1], "", err)
		}

		// Unrecognized flag error — wrap so it's still typed
		return structclierrors.NewFlagError(structclierrors.FlagErrorInvalidValue, "", "", err)
	})
}

// StructuredError is the JSON object written to stderr by HandleError.
//
// Every field is optional except Error, ExitCode, and Message.
// Agents parse this to decide whether to self-correct, fix the environment, or report to a human.
type StructuredError struct {
	Error    string `json:"error"`
	ExitCode int    `json:"exit_code"`
	Message  string `json:"message"`

	// Input error fields
	Flag      string   `json:"flag,omitempty"`
	Got       string   `json:"got,omitempty"`
	Expected  string   `json:"expected,omitempty"`
	Command   string   `json:"command,omitempty"`
	Hint      string   `json:"hint,omitempty"`
	Available []string `json:"available,omitempty"`

	// Validation fields
	Violations []Violation `json:"violations,omitempty"`

	// Config fields
	ConfigFile string `json:"config_file,omitempty"`
	Key        string `json:"key,omitempty"`

	// Environment variable fields
	EnvVar string `json:"env_var,omitempty"`
}

// Violation represents a single validation failure for a field.
type Violation struct {
	Field   string `json:"field"`
	Rule    string `json:"rule,omitempty"`
	Param   string `json:"param,omitempty"`
	Value   any    `json:"value,omitempty"`
	Message string `json:"message"`
}

// HandleError classifies err, writes a JSON StructuredError to w, and returns a semantic exit code.
//
// The cmd parameter must be the command where the error originated — not the root command.
// This is because HandleError looks up flag metadata (type, enum values, env var bindings)
// from cmd's flag annotations to produce accurate error details. If the root command is
// passed for a subcommand error, the metadata lookup yields empty results and the output
// is degraded (no expected type, no enum check, no env var attribution).
//
// Use [ExecuteOrExit] to get this right automatically — it uses cobra's ExecuteC to obtain
// the correct command. If calling HandleError directly, use [cobra.Command.ExecuteC]:
//
//	cmd, err := rootCmd.ExecuteC()
//	if err != nil {
//	    os.Exit(structcli.HandleError(cmd, err, os.Stderr))
//	}
//
// HandleError has no side effects beyond reading the current process environment to improve
// source attribution and writing the structured error JSON to w.
//
// If err is nil, HandleError returns exitcode.OK and writes nothing.
func HandleError(cmd *cobra.Command, err error, w io.Writer) int {
	if err == nil {
		return exitcode.OK
	}

	se := classify(cmd, err)

	out, marshalErr := json.Marshal(se)
	if marshalErr != nil {
		// Last resort: write the original error as-is.
		fmt.Fprintf(w, `{"error":"error","exit_code":1,"message":%q}`+"\n", err.Error())

		return exitcode.Error
	}
	fmt.Fprintln(w, string(out))

	return se.ExitCode
}

// ExecuteOrExit is a convenience wrapper around ExecuteC for the common main() pattern.
// On error it writes structured JSON to stderr and exits with a semantic exit code.
// On success it exits 0.
//
//	func main() {
//	    structcli.ExecuteOrExit(buildMyCLI())
//	}
func ExecuteOrExit(cmd *cobra.Command) {
	c, err := ExecuteC(cmd)
	if err != nil {
		os.Exit(HandleError(c, err, os.Stderr))
	}
}

// classify inspects err and produces a StructuredError with the correct error code and exit code.
func classify(cmd *cobra.Command, err error) *StructuredError {
	errMsg := err.Error()
	cmdPath := commandPath(cmd)

	// 1. structcli typed errors (errors.As)

	// ValidationError from ValidatableOptions
	var validationErr *structclierrors.ValidationError
	if errors.As(err, &validationErr) {
		return classifyValidation(cmd, cmdPath, validationErr)
	}

	// InputError from structcli
	var inputErr *structclierrors.InputError
	if errors.As(err, &inputErr) {
		return &StructuredError{
			Error:    "invalid_input",
			ExitCode: exitcode.InvalidFlagValue,
			Command:  cmdPath,
			Message:  errMsg,
		}
	}

	// EnvOnlyCLIUsageError from Unmarshal's post-parse check
	var envOnlyCLIErr *structclierrors.EnvOnlyCLIUsageError
	if errors.As(err, &envOnlyCLIErr) {
		se := &StructuredError{
			Error:    "env_only_cli_usage",
			ExitCode: exitcode.InvalidFlagValue,
			Command:  cmdPath,
			Message:  errMsg,
		}
		if len(envOnlyCLIErr.FlagNames) == 1 {
			se.Flag = envOnlyCLIErr.FlagNames[0]
			envVars := flagEnvVars(cmd, envOnlyCLIErr.FlagNames[0])
			if len(envVars) > 0 {
				se.Hint = fmt.Sprintf("set %s instead", envVars[0])
				se.EnvVar = envVars[0]
			}
		}

		return se
	}

	// 2. Typed flag errors from SetupFlagErrors (errors.As — no regex needed)
	// FlagError carries only the flag name, value, and kind. Metadata enrichment
	// (expected type, enum values, env vars) happens here via the same code path
	// as the regex fallback — using the correct command from ExecuteC().
	var flagErr *structclierrors.FlagError
	if errors.As(err, &flagErr) {
		switch flagErr.Kind {
		case structclierrors.FlagErrorInvalidValue:
			return classifyInvalidArg(cmd, cmdPath, flagErr.Value, flagErr.FlagName, errMsg)
		case structclierrors.FlagErrorUnknown:
			return &StructuredError{
				Error:    "unknown_flag",
				ExitCode: exitcode.UnknownFlag,
				Flag:     flagErr.FlagName,
				Command:  cmdPath,
				Message:  errMsg,
			}
		}
	}

	// 3. Cobra string-pattern errors (fallback when SetupFlagErrors is not active)

	// Required flag(s) not set
	if m := reRequiredFlags.FindStringSubmatch(errMsg); m != nil {
		return classifyMissingRequired(cmd, cmdPath, m[1], errMsg)
	}

	// Invalid argument for flag
	if m := reInvalidArg.FindStringSubmatch(errMsg); m != nil {
		return classifyInvalidArg(cmd, cmdPath, m[1], m[2], errMsg)
	}

	// Unknown flag
	if m := reUnknownFlag.FindStringSubmatch(errMsg); m != nil {
		return &StructuredError{
			Error:    "unknown_flag",
			ExitCode: exitcode.UnknownFlag,
			Flag:     m[1],
			Command:  cmdPath,
			Message:  errMsg,
		}
	}

	// Unknown command (no typed error possible — cobra creates these inline)
	if m := reUnknownCommand.FindStringSubmatch(errMsg); m != nil {
		return classifyUnknownCommand(cmd, m[1], cmdPath, errMsg)
	}

	// 3. Config errors

	// Unknown config keys
	if m := reConfigUnknownKeys.FindStringSubmatch(errMsg); m != nil {
		keys := strings.Split(m[1], ", ")
		return &StructuredError{
			Error:    "config_unknown_key",
			ExitCode: exitcode.ConfigUnknownKey,
			Key:      keys[0],
			Command:  cmdPath,
			Message:  errMsg,
		}
	}

	// Config parse / file errors
	if strings.Contains(errMsg, "config file") || strings.Contains(errMsg, "While parsing config") {
		if strings.Contains(errMsg, "Not Found") || strings.Contains(errMsg, "no such file") {
			return &StructuredError{
				Error:    "config_not_found",
				ExitCode: exitcode.ConfigNotFound,
				Command:  cmdPath,
				Message:  errMsg,
			}
		}

		return &StructuredError{
			Error:    "config_parse_error",
			ExitCode: exitcode.ConfigParseError,
			Command:  cmdPath,
			Message:  errMsg,
		}
	}

	// 4. Unmarshal/decode errors (from structcli.Unmarshal via mapstructure)
	// Pattern: "couldn't unmarshal config to options: decoding failed ... 'Field' cannot parse value as 'type'"
	if strings.Contains(errMsg, "unmarshal") && strings.Contains(errMsg, "decoding failed") {
		return classifyUnmarshalError(cmd, cmdPath, errMsg)
	}

	// 5. Generic fallback
	return &StructuredError{
		Error:    "error",
		ExitCode: exitcode.Error,
		Command:  cmdPath,
		Message:  errMsg,
	}
}

// classifyValidation builds a StructuredError from a ValidationError.
// It uses Details() to extract structured field/rule/value information and
// resolves Go struct field names to CLI flag names via the command's annotations.
func classifyValidation(cmd *cobra.Command, cmdPath string, ve *structclierrors.ValidationError) *StructuredError {
	details := ve.Details()
	violations := make([]Violation, 0, len(details))
	for _, d := range details {
		field := d.Field
		// Try to resolve Go struct field name to CLI flag name
		if d.StructField != "" {
			if flagName := findFlagForField(cmd, d.StructField); flagName != "" {
				field = flagName
			}
		}
		// Fall back to Field if StructField resolution failed
		if field == "" && d.Field != "" {
			if flagName := findFlagForField(cmd, d.Field); flagName != "" {
				field = flagName
			} else {
				field = d.Field
			}
		}

		violations = append(violations, Violation{
			Field:   field,
			Rule:    d.Rule,
			Param:   d.Param,
			Value:   d.Value,
			Message: d.Message,
		})
	}

	// Fallback: if Details() returned nothing but there are errors, use the raw error messages
	if len(violations) == 0 {
		for _, e := range ve.Errors {
			if e == nil {
				continue
			}
			violations = append(violations, Violation{
				Message: e.Error(),
			})
		}
	}

	return &StructuredError{
		Error:      "validation_failed",
		ExitCode:   exitcode.ValidationFailed,
		Command:    cmdPath,
		Violations: violations,
		Message:    ve.Error(),
	}
}

// classifyMissingRequired handles the "required flag(s) ... not set" cobra error.
// It keeps the primary classification focused on the missing required flag while
// optionally enriching the hint with env fallback or validation context.
func classifyMissingRequired(cmd *cobra.Command, cmdPath, flagList, errMsg string) *StructuredError {
	// Parse flag names from: "port" or "port", "host"
	flagNames := parseQuotedList(flagList)

	// For single-flag errors, provide enriched output with remedy hints.
	if len(flagNames) == 1 {
		flagName := flagNames[0]

		// Env-only flags get a distinct error classification and exit code.
		if flagIsEnvOnly(cmd, flagName) {
			envVars := flagEnvVars(cmd, flagName)
			se := &StructuredError{
				Error:    "missing_required_env",
				ExitCode: exitcode.EnvMissingRequired,
				Command:  cmdPath,
				Message:  structclierrors.NewMissingRequiredEnvError(flagName, envVars).Error(),
			}
			if len(envVars) > 0 {
				se.EnvVar = envVars[0]
				se.Hint = fmt.Sprintf("set %s", strings.Join(envVars, " or "))
			}

			return se
		}

		// Build hint from env fallbacks and validation rules without changing the
		// top-level classification away from the missing required flag.
		var hint string
		envVars := flagEnvVars(cmd, flagName)
		envSet := false
		for _, ev := range envVars {
			if _, ok := os.LookupEnv(ev); ok {
				envSet = true
				break
			}
		}
		if len(envVars) > 0 && !envSet {
			hint = fmt.Sprintf("use --%s <value> or set %s", flagName, envVars[0])
		}
		if rules := flagValidateRules(cmd, flagName); strings.Contains(rules, "required") {
			if hint != "" {
				hint += " (required by validation)"
			} else {
				hint = fmt.Sprintf("--%s is required by validation", flagName)
			}
		}

		se := &StructuredError{
			Error:    "missing_required_flag",
			ExitCode: exitcode.MissingRequiredFlag,
			Flag:     flagName,
			Command:  cmdPath,
			Message:  errMsg,
		}
		if hint != "" {
			se.Hint = hint
		}

		return se
	}

	// Multiple missing flags — no per-flag enrichment
	return &StructuredError{
		Error:    "missing_required_flag",
		ExitCode: exitcode.MissingRequiredFlag,
		Command:  cmdPath,
		Message:  errMsg,
	}
}

// classifyInvalidArg handles the `invalid argument "val" for "flags" flag:` cobra error.
// It does source attribution to determine if the bad value came from the CLI, an env var, or config.
func classifyInvalidArg(cmd *cobra.Command, cmdPath, gotValue, flagSpec, errMsg string) *StructuredError {
	// flagSpec is like "-p, --port" or "--port" — extract the long name
	flagName := extractLongFlagName(flagSpec)
	expected := flagType(cmd, flagName)

	// Check if the flag has enum annotations — if so, this might be an enum violation
	if enumVals := flagEnumValues(cmd, flagName); len(enumVals) > 0 {
		if !contains(enumVals, gotValue) {
			return &StructuredError{
				Error:     "invalid_flag_enum",
				ExitCode:  exitcode.InvalidFlagEnum,
				Flag:      flagName,
				Got:       gotValue,
				Expected:  strings.Join(enumVals, ", "),
				Available: enumVals,
				Command:   cmdPath,
				Message:   errMsg,
			}
		}
	}

	// Source attribution: where did the bad value come from?

	// 1. Check if flag was explicitly set on CLI
	if cmd.Flags().Lookup(flagName) != nil && cmd.Flags().Lookup(flagName).Changed {
		return &StructuredError{
			Error:    "invalid_flag_value",
			ExitCode: exitcode.InvalidFlagValue,
			Flag:     flagName,
			Got:      gotValue,
			Expected: expected,
			Command:  cmdPath,
			Message:  errMsg,
		}
	}

	// 2. Check env vars
		if envVars := flagEnvVars(cmd, flagName); len(envVars) > 0 {
			for _, ev := range envVars {
				if val, ok := os.LookupEnv(ev); ok {
					return &StructuredError{
						Error:    "env_invalid_value",
						ExitCode: exitcode.EnvInvalidValue,
						EnvVar:   ev,
						Flag:     flagName,
						Got:      val,
						Expected: expected,
						Command:  cmdPath,
						Message:  fmt.Sprintf("env var %s: invalid value %q for flag %q (expected %s)", ev, val, flagName, expected),
					}
				}
			}
		}

	// 3. Default: attribute to CLI flag (cobra reports it this way)
	return &StructuredError{
		Error:    "invalid_flag_value",
		ExitCode: exitcode.InvalidFlagValue,
		Flag:     flagName,
		Got:      gotValue,
		Expected: expected,
		Command:  cmdPath,
		Message:  errMsg,
	}
}

// classifyUnknownCommand builds a StructuredError for an unknown subcommand.
func classifyUnknownCommand(cmd *cobra.Command, got, cmdPath, errMsg string) *StructuredError {
	var available []string
	root := cmd.Root()
	for _, sub := range root.Commands() {
		if !sub.IsAvailableCommand() && sub.Name() != "help" {
			continue
		}
		available = append(available, sub.Name())
	}

	return &StructuredError{
		Error:     "unknown_command",
		ExitCode:  exitcode.UnknownCommand,
		Got:       got,
		Command:   cmdPath,
		Available: available,
		Message:   errMsg,
	}
}

// classifyUnmarshalError handles errors from structcli.Unmarshal (mapstructure decode failures).
// These typically happen when an env var or config value has the wrong type for a field.
// It does source attribution to distinguish env var errors from config errors.
func classifyUnmarshalError(cmd *cobra.Command, cmdPath, errMsg string) *StructuredError {
	fieldName, gotValue, expectedType := parseDecodeError(errMsg)

	if fieldName == "" {
		// Can't parse the specific field — treat as a config-origin decode failure.
		return &StructuredError{
			Error:    "config_invalid_value",
			ExitCode: exitcode.ConfigInvalidValue,
			Command:  cmdPath,
			Message:  errMsg,
		}
	}

	// Look for a flag matching this field name
	flagName := findFlagForField(cmd, fieldName)

	// Enrich expected type from the flag if not already known
	if expectedType == "" && flagName != "" {
		expectedType = flagType(cmd, flagName)
	}

	// Check if the flag has enum annotations — if so, this might be an enum violation
	if flagName != "" && gotValue != "" {
		if enumVals := flagEnumValues(cmd, flagName); len(enumVals) > 0 {
			if !contains(enumVals, gotValue) {
				return &StructuredError{
					Error:     "invalid_flag_enum",
					ExitCode:  exitcode.InvalidFlagEnum,
					Flag:      flagName,
					Got:       gotValue,
					Expected:  strings.Join(enumVals, ", "),
					Available: enumVals,
					Command:   cmdPath,
					Message:   errMsg,
				}
			}
		}
	}

	// Source attribution: check if the bad value came from an env var
	if flagName != "" {
			if envVars := flagEnvVars(cmd, flagName); len(envVars) > 0 {
				for _, ev := range envVars {
					if val, ok := os.LookupEnv(ev); ok {
						if gotValue == "" {
							gotValue = val
						}

						return &StructuredError{
							Error:    "env_invalid_value",
							ExitCode: exitcode.EnvInvalidValue,
							EnvVar:   ev,
							Flag:     flagName,
							Got:      gotValue,
							Expected: expectedType,
							Command:  cmdPath,
							Message:  fmt.Sprintf("env var %s: invalid value %q for flag %q (expected %s)", ev, gotValue, flagName, expectedType),
						}
					}
				}
			}
	}

	// Not from env — could be from config or default
	return &StructuredError{
		Error:    "config_invalid_value",
		ExitCode: exitcode.ConfigInvalidValue,
		Flag:     flagName,
		Got:      gotValue,
		Expected: expectedType,
		Command:  cmdPath,
		Message:  errMsg,
	}
}

// parseDecodeError extracts field name, bad value, and expected type from mapstructure errors.
// It tries multiple patterns since mapstructure produces different formats for different types.
func parseDecodeError(errMsg string) (fieldName, gotValue, expectedType string) {
	// Pattern 1: 'Port' cannot parse value as 'int': ...
	if m := reDecodeFieldError.FindStringSubmatch(errMsg); m != nil {
		return m[1], "", m[2]
	}

	// Pattern 2: 'LogLevel' invalid string for zapcore.Level 'bogus': ...
	if m := reDecodeFieldInvalid.FindStringSubmatch(errMsg); m != nil {
		return m[1], m[2], ""
	}

	// Pattern 3: at minimum, extract the field name from 'FieldName'
	if m := reDecodeFieldName.FindStringSubmatch(errMsg); m != nil {
		return m[1], "", ""
	}

	return "", "", ""
}

// flagType returns the type string of a flag, or empty string if not found.
func flagType(cmd *cobra.Command, flagName string) string {
	f := cmd.Flags().Lookup(flagName)
	if f == nil {
		f = cmd.InheritedFlags().Lookup(flagName)
	}
	if f == nil {
		return ""
	}

	return f.Value.Type()
}

// findFlagForField finds a flag name that matches a struct field name.
// It checks: exact flag name match, then field path annotation match.
// Field names from mapstructure errors are Go struct field names (eg. "LogLevel"),
// while flag names are kebab-case (eg. "level" or "log-level").
func findFlagForField(cmd *cobra.Command, fieldName string) string {
	lowerField := strings.ToLower(fieldName)
	var found string
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if found != "" {
			return
		}
		// Direct match: flag name == field name (case-insensitive)
		if strings.EqualFold(f.Name, lowerField) {
			found = f.Name

			return
		}
		// Path annotation match: structcli stores the lowercased field path
		if f.Annotations != nil {
			if paths, ok := f.Annotations[flagPathAnnotation]; ok && len(paths) > 0 {
				if strings.EqualFold(paths[0], lowerField) {
					found = f.Name

					return
				}
			}
		}
	})

	return found
}

// flagEnvVars returns the env var names bound to a flag, or nil if none.
func flagIsEnvOnly(cmd *cobra.Command, flagName string) bool {
	f := cmd.Flags().Lookup(flagName)
	if f == nil {
		f = cmd.InheritedFlags().Lookup(flagName)
	}
	if f == nil || f.Annotations == nil {
		return false
	}
	_, ok := f.Annotations[internalenv.FlagEnvOnlyAnnotation]

	return ok
}

func flagEnvVars(cmd *cobra.Command, flagName string) []string {
	f := cmd.Flags().Lookup(flagName)
	if f == nil {
		// Try inherited (persistent) flags
		f = cmd.InheritedFlags().Lookup(flagName)
	}
	if f == nil || f.Annotations == nil {
		return nil
	}

	envs, ok := f.Annotations[internalenv.FlagAnnotation]
	if !ok || len(envs) == 0 {
		return nil
	}

	return envs
}

// flagEnumValues returns the allowed enum values for a flag, or nil if none.
func flagEnumValues(cmd *cobra.Command, flagName string) []string {
	f := cmd.Flags().Lookup(flagName)
	if f == nil {
		f = cmd.InheritedFlags().Lookup(flagName)
	}
	if f == nil || f.Annotations == nil {
		return nil
	}

	vals, ok := f.Annotations[flagEnumAnnotation]
	if !ok || len(vals) == 0 {
		return nil
	}

	return vals
}

// flagValidateRules returns the validation rules for a flag, or empty string if none.
func flagValidateRules(cmd *cobra.Command, flagName string) string {
	f := cmd.Flags().Lookup(flagName)
	if f == nil {
		f = cmd.InheritedFlags().Lookup(flagName)
	}
	if f == nil || f.Annotations == nil {
		return ""
	}

	vals, ok := f.Annotations[flagValidateAnnotation]
	if !ok || len(vals) == 0 {
		return ""
	}

	return vals[0]
}

// contains checks if a string is in a slice.
func contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}

	return false
}

// extractLongFlagName extracts the long flag name from cobra's flag spec.
// Input: "-p, --port" or "--port" → "port"
func extractLongFlagName(spec string) string {
	parts := strings.Split(spec, "--")
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[len(parts)-1])
	}

	return strings.TrimLeft(strings.TrimSpace(spec), "-")
}

// parseQuotedList parses cobra's quoted flag list: `"port"` or `"port", "host"`.
func parseQuotedList(s string) []string {
	var result []string
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"`)
		if p != "" {
			result = append(result, p)
		}
	}

	return result
}

// commandPath returns the full command path for inclusion in errors.
func commandPath(cmd *cobra.Command) string {
	return cmd.CommandPath()
}
