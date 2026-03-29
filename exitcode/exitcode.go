// Package exitcode defines semantic exit codes for structcli-powered CLIs.
//
// Exit codes are grouped into ranges that form a decision tree for AI agents:
//
//	exit_code == 0  → success
//	exit_code 1–9   → runtime error → report to human
//	exit_code 10–19 → input error → self-correct from error JSON → retry
//	exit_code 20–29 → config/env error → fix environment → retry
//
// These codes are returned by [structcli.HandleError] and included
// in the structured JSON error output as the "exit_code" field.
package exitcode

// Runtime errors (1–9): not the agent's fault — report, don't retry.
const (
	// OK indicates successful execution.
	OK = 0

	// Error is the fallback for unclassified runtime errors.
	Error = 1

	// PermissionDenied indicates a filesystem or network permission failure.
	PermissionDenied = 2

	// Timeout indicates the operation exceeded its time limit.
	Timeout = 3

	// Interrupted indicates the process received SIGINT or SIGTERM.
	Interrupted = 4
)

// Input errors (10–19): the agent provided bad input — self-correct and retry.
const (
	// MissingRequiredFlag indicates a required input was missing at command
	// execution time. Structured errors may include hints about env fallbacks,
	// but the primary classification remains a missing required flag/input.
	MissingRequiredFlag = 10

	// InvalidFlagValue indicates a flag value has the wrong type or format
	// (eg. "abc" for an int flag).
	InvalidFlagValue = 11

	// UnknownFlag indicates the flag does not exist on the command.
	UnknownFlag = 12

	// ValidationFailed indicates one or more custom validation rules failed
	// (from ValidatableOptions). The structured error JSON includes a
	// "violations" array with per-field details.
	ValidationFailed = 13

	// UnknownCommand indicates the subcommand does not exist.
	// The structured error JSON includes an "available" array.
	UnknownCommand = 14

	// InvalidFlagEnum indicates the value is not in the allowed enum set.
	InvalidFlagEnum = 15
)

// Configuration and environment errors (20–29): the environment is wrong — fix it, then retry.
const (
	// ConfigParseError indicates the config file exists but is malformed
	// (invalid YAML, JSON, or TOML syntax).
	ConfigParseError = 20

	// ConfigUnknownKey indicates the config file contains an unrecognized key.
	// The structured error JSON includes an "available" array of valid keys.
	ConfigUnknownKey = 21

	// ConfigInvalidValue indicates a config key has the wrong type or format.
	ConfigInvalidValue = 22

	// ConfigNotFound indicates the path passed via --config does not exist.
	ConfigNotFound = 23

	// EnvInvalidValue indicates an environment variable is set but has the
	// wrong format or type for its target flag.
	EnvInvalidValue = 25

	// Deprecated: reserved for compatibility. HandleError currently reports
	// missing required inputs as MissingRequiredFlag and may include env
	// fallback hints in the structured error payload instead.
	EnvMissingRequired = 26
)

// Category names returned by [Category].
const (
	CategoryOK      = "ok"
	CategoryRuntime = "runtime"
	CategoryInput   = "input"
	CategoryConfig  = "config"
)

// Category returns the error category for a given exit code.
//
// Agents use this to decide their top-level strategy:
//   - "ok": success — proceed
//   - "input": bad input — self-correct from the error JSON and retry
//   - "config": environment problem — fix config/env vars, then retry
//   - "runtime": not the agent's fault — report to human
func Category(code int) string {
	switch {
	case code == OK:
		return CategoryOK
	case code >= 10 && code <= 19:
		return CategoryInput
	case code >= 20 && code <= 29:
		return CategoryConfig
	default:
		return CategoryRuntime
	}
}

// IsRetryable returns true if the error category suggests the agent
// can self-correct and retry (input or config/env errors).
func IsRetryable(code int) bool {
	cat := Category(code)
	return cat == CategoryInput || cat == CategoryConfig
}
