package errors

import (
	"errors"
	"fmt"
	"strings"
)

// fieldErrorInfo is satisfied by validator.FieldError from go-playground/validator
// and any other validation library that provides structured field error information.
type fieldErrorInfo interface {
	Field() string
	StructField() string
	Tag() string
	Param() string
	Value() interface{}
}

// ValidationError wraps multiple validation errors that occurred during ValidatableOptions unmarshalling.
type ValidationError struct {
	ContextName string
	Errors      []error
}

func (e *ValidationError) Error() string {
	var sb strings.Builder
	if e.ContextName != "" {
		sb.WriteString(fmt.Sprintf("invalid options for %s", e.ContextName))
	} else {
		sb.WriteString("invalid options")
	}
	if len(e.Errors) >= 1 {
		sb.WriteString(":")
	}

	for _, err := range e.Errors {
		if err == nil {
			continue
		}
		sb.WriteString("\n       ")
		sb.WriteString(err.Error())
	}

	return sb.String()
}

// ValidationDetail holds structured information extracted from a single validation error.
type ValidationDetail struct {
	Field       string `json:"field,omitempty"`
	StructField string `json:"structField,omitempty"`
	Rule        string `json:"rule,omitempty"`
	Param       string `json:"param,omitempty"`
	Value       any    `json:"value,omitempty"`
	Message     string `json:"message"`
}

// Details extracts structured information from each inner error.
// For errors implementing fieldErrorInfo (e.g., validator.FieldError), it extracts Field, Rule (tag), Param, and Value.
// For all other errors, it falls back to populating only the Message field.
// Returns nil when Errors is nil or empty.
func (e *ValidationError) Details() []ValidationDetail {
	if len(e.Errors) == 0 {
		return nil
	}

	details := make([]ValidationDetail, 0, len(e.Errors))
	for _, err := range e.Errors {
		if err == nil {
			continue
		}
		var fe fieldErrorInfo
		if errors.As(err, &fe) {
			details = append(details, ValidationDetail{
				Field:       fe.Field(),
				StructField: fe.StructField(),
				Rule:        fe.Tag(),
				Param:       fe.Param(),
				Value:       fe.Value(),
				Message:     err.Error(),
			})
		} else {
			details = append(details, ValidationDetail{
				Message: err.Error(),
			})
		}
	}

	return details
}

// Unwrap returns the inner errors so that errors.Is and errors.As
// traverse into individual validation failures.
//
// The returned slice is the live internal slice. Callers that need
// mutation-safe access should use [ValidationError.UnderlyingErrors] instead.
func (e *ValidationError) Unwrap() []error {
	return e.Errors
}

// UnderlyingErrors returns a defensive copy of the individual validation errors.
func (e *ValidationError) UnderlyingErrors() []error {
	if e.Errors == nil {
		return nil
	}

	// Return a copy to prevent mutations
	result := make([]error, len(e.Errors))
	copy(result, e.Errors)

	return result
}

// These are all DefinitionError
var (
	ErrInvalidBooleanTag = errors.New("invalid boolean tag value")
	ErrInvalidFlagEnvTag = errors.New("invalid flagenv tag value")
	ErrInvalidShorthand  = errors.New("invalid shorthand flag")
	ErrInvalidTagUsage   = errors.New("invalid tag usage")
	ErrConflictingTags   = errors.New("conflicting struct tags")
	ErrUnsupportedType   = errors.New("unsupported field type")
	ErrDuplicateFlag     = errors.New("duplicate flag name")
	ErrInvalidFlagName   = errors.New("invalid flag name")
)

// DefinitionError represents an error that occurred while processing a struct field's tags at definition time.
type DefinitionError interface {
	error
	Field() string
}

// DuplicateFlagError represents a flag name that is already in use.
type DuplicateFlagError struct {
	FlagName          string
	NewFieldPath      string
	ExistingFieldPath string
}

func (e *DuplicateFlagError) Error() string {
	return fmt.Sprintf("field '%s': flag name '%s' is already in use by field '%s'", e.NewFieldPath, e.FlagName, e.ExistingFieldPath)
}

func (e *DuplicateFlagError) Field() string {
	return e.NewFieldPath
}

func (e *DuplicateFlagError) Unwrap() error {
	return ErrDuplicateFlag
}

// InvalidBooleanTagError represents an invalid boolean value in struct tags
type InvalidBooleanTagError struct {
	FieldName string
	TagName   string
	TagValue  string
}

func (e *InvalidBooleanTagError) Error() string {
	return fmt.Sprintf("field '%s': tag '%s=%s': invalid boolean value", e.FieldName, e.TagName, e.TagValue)
}

func (e *InvalidBooleanTagError) Field() string {
	return e.FieldName
}

func (e *InvalidBooleanTagError) Unwrap() error {
	return ErrInvalidBooleanTag
}

// InvalidFlagEnvTagError represents an invalid value for the flagenv tag.
type InvalidFlagEnvTagError struct {
	FieldName string
	TagValue  string
}

func (e *InvalidFlagEnvTagError) Error() string {
	return fmt.Sprintf("field '%s': tag 'flagenv=%s': invalid value (expected true, false, or only)", e.FieldName, e.TagValue)
}

func (e *InvalidFlagEnvTagError) Field() string {
	return e.FieldName
}

func (e *InvalidFlagEnvTagError) Unwrap() error {
	return ErrInvalidFlagEnvTag
}

func NewInvalidFlagEnvTagError(fieldName, tagValue string) error {
	return &InvalidFlagEnvTagError{
		FieldName: fieldName,
		TagValue:  tagValue,
	}
}

// InvalidShorthandError represents an invalid shorthand flag specification
type InvalidShorthandError struct {
	FieldName string
	Shorthand string
}

func (e *InvalidShorthandError) Error() string {
	return fmt.Sprintf("field '%s': shorthand flag '%s' must be a single character", e.FieldName, e.Shorthand)
}

func (e *InvalidShorthandError) Field() string {
	return e.FieldName
}

func (e *InvalidShorthandError) Unwrap() error {
	return ErrInvalidShorthand
}

// InvalidTagUsageError represents invalid tag usages
type InvalidTagUsageError struct {
	FieldName string
	TagName   string
	Message   string
}

func (e *InvalidTagUsageError) Error() string {
	return fmt.Sprintf("field '%s': invalid usage of tag '%s': %s", e.FieldName, e.TagName, e.Message)
}

func (e *InvalidTagUsageError) Field() string {
	return e.FieldName
}

func (e *InvalidTagUsageError) Unwrap() error {
	return ErrInvalidTagUsage
}

// ConflictingTagsError represents conflicting struct tag values
type ConflictingTagsError struct {
	FieldName       string
	Message         string
	ConflictingTags []string
}

func (e *ConflictingTagsError) Error() string {
	return fmt.Sprintf(
		"field '%s': conflicting tags [%s]: %s",
		e.FieldName,
		strings.Join(e.ConflictingTags, ", "),
		e.Message,
	)
}

func (e *ConflictingTagsError) Field() string {
	return e.FieldName
}
func (e *ConflictingTagsError) Unwrap() error {
	return ErrConflictingTags
}

// UnsupportedTypeError represents an unsupported field type
type UnsupportedTypeError struct {
	FieldName string
	FieldType string
	Message   string
}

func (e *UnsupportedTypeError) Error() string {
	return fmt.Sprintf("field '%s': unsupported type '%s': %s", e.FieldName, e.FieldType, e.Message)
}

func (e *UnsupportedTypeError) Field() string {
	return e.FieldName
}

func (e *UnsupportedTypeError) Unwrap() error {
	return ErrUnsupportedType
}

// InvalidFlagNameError represents an invalid flag name
type InvalidFlagNameError struct {
	FieldName string
	FlagName  string
}

func (e *InvalidFlagNameError) Error() string {
	return fmt.Sprintf("field '%s': generated flag name '%s' is invalid. Use only alphanumeric characters, dashes, and dots.", e.FieldName, e.FlagName)
}

func (e *InvalidFlagNameError) Field() string {
	return e.FieldName
}

func (e *InvalidFlagNameError) Unwrap() error {
	return ErrInvalidFlagName
}

func NewInvalidFlagNameError(fieldName, flagName string) error {
	return &InvalidFlagNameError{
		FieldName: fieldName,
		FlagName:  flagName,
	}
}

func NewDuplicateFlagError(flagName, newFieldPath, existingFieldPath string) error {
	return &DuplicateFlagError{
		FlagName:          flagName,
		NewFieldPath:      newFieldPath,
		ExistingFieldPath: existingFieldPath,
	}
}

func NewInvalidBooleanTagError(fieldName, tagName, tagValue string) error {
	return &InvalidBooleanTagError{
		FieldName: fieldName,
		TagName:   tagName,
		TagValue:  tagValue,
	}
}

func NewInvalidShorthandError(fieldName, shorthand string) error {
	return &InvalidShorthandError{
		FieldName: fieldName,
		Shorthand: shorthand,
	}
}

func NewInvalidTagUsageError(fieldName, tagName, message string) error {
	return &InvalidTagUsageError{
		FieldName: fieldName,
		TagName:   tagName,
		Message:   message,
	}
}

func NewConflictingTagsError(fieldName string, tags []string, message string) error {
	return &ConflictingTagsError{
		FieldName:       fieldName,
		ConflictingTags: tags,
		Message:         message,
	}
}

func NewUnsupportedTypeError(fieldName, fieldType, message string) error {
	return &UnsupportedTypeError{
		FieldName: fieldName,
		FieldType: fieldType,
		Message:   message,
	}
}

var ErrMissingRequiredEnv = errors.New("missing required environment variable")

var ErrEnvOnlyCLIUsage = errors.New("env-only flag set via CLI")

// EnvOnlyCLIUsageError represents an attempt to set an env-only flag via the CLI.
type EnvOnlyCLIUsageError struct {
	FlagNames []string
}

func (e *EnvOnlyCLIUsageError) Error() string {
	return fmt.Sprintf("flag(s) %s can only be set via environment variable, not --flag",
		strings.Join(e.FlagNames, ", "))
}

func (e *EnvOnlyCLIUsageError) Unwrap() error {
	return ErrEnvOnlyCLIUsage
}

// MissingRequiredEnvError represents an env-only field that was not set.
type MissingRequiredEnvError struct {
	FieldName string
	EnvVars   []string
}

func (e *MissingRequiredEnvError) Error() string {
	return fmt.Sprintf("required environment variable(s) not set: %s (for field '%s')",
		strings.Join(e.EnvVars, " or "), e.FieldName)
}

func (e *MissingRequiredEnvError) Field() string {
	return e.FieldName
}

func (e *MissingRequiredEnvError) Unwrap() error {
	return ErrMissingRequiredEnv
}

func NewMissingRequiredEnvError(fieldName string, envVars []string) error {
	return &MissingRequiredEnvError{
		FieldName: fieldName,
		EnvVars:   envVars,
	}
}

var ErrInputValue = errors.New("invalid input value")

// InputError represents an invalid input value for flag definition
type InputError struct {
	InputType string
	Message   string
}

func (e *InputError) Error() string {
	return fmt.Sprintf("invalid input value of type '%s': %s", e.InputType, e.Message)
}

func (e *InputError) Unwrap() error {
	return ErrInputValue
}

// NewInputError creates an InputError.
func NewInputError(inputType, message string) error {
	return &InputError{
		InputType: inputType,
		Message:   message,
	}
}

// FlagError represents a flag parsing error intercepted by [SetupFlagErrors].
//
// It carries only what's needed to identify the error: flag name, bad value,
// and error kind. Metadata enrichment (expected type, enum values, env vars)
// happens at classification time in [HandleError], which receives the correct
// subcommand from [ExecuteC].
type FlagError struct {
	FlagName string        // the flag name (eg. "port", "level")
	Value    string        // the value that was provided (may be empty for unknown flags)
	Kind     FlagErrorKind // what kind of flag error
	Cause    error         // the original pflag error
}

// FlagErrorKind distinguishes between flag error types.
type FlagErrorKind int

const (
	// FlagErrorInvalidValue indicates a flag received a value of the wrong type/format.
	FlagErrorInvalidValue FlagErrorKind = iota
	// FlagErrorUnknown indicates the flag does not exist on the command.
	FlagErrorUnknown
)

func (e *FlagError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}

	switch e.Kind {
	case FlagErrorUnknown:
		return fmt.Sprintf("unknown flag: --%s", e.FlagName)
	case FlagErrorInvalidValue:
		return fmt.Sprintf("invalid value %q for flag --%s", e.Value, e.FlagName)
	default:
		return fmt.Sprintf("flag error: --%s", e.FlagName)
	}
}

func (e *FlagError) Unwrap() error {
	return e.Cause
}

// NewFlagError creates a FlagError.
func NewFlagError(kind FlagErrorKind, flagName, value string, cause error) *FlagError {
	return &FlagError{
		FlagName: flagName,
		Value:    value,
		Kind:     kind,
		Cause:    cause,
	}
}
