package internalvalidation

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	structclierrors "github.com/leodido/structcli/errors"
	internalenv "github.com/leodido/structcli/internal/env"
	internalhooks "github.com/leodido/structcli/internal/hooks"
	internalpath "github.com/leodido/structcli/internal/path"
	internalreflect "github.com/leodido/structcli/internal/reflect"
	internalscope "github.com/leodido/structcli/internal/scope"
	internaltag "github.com/leodido/structcli/internal/tag"
	"github.com/spf13/cobra"
)

// IsValidBoolTag validates that a struct tag contains a valid boolean value
func IsValidBoolTag(fieldName, tagName, tagValue string) (*bool, error) {
	if tagValue == "" {
		return nil, nil
	}
	val, err := strconv.ParseBool(tagValue)
	if err != nil {
		return nil, structclierrors.NewInvalidBooleanTagError(fieldName, tagName, tagValue)
	}

	return &val, nil
}

// Struct checks the coherence of definitions in the given struct.
func Struct(c *cobra.Command, o any) error {
	val, err := internalreflect.GetValidValue(o)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	s := internalscope.Get(c)

	typeName := val.Type().Name()
	if err := Fields(val, typeName, s); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// Fields recursively validates the struct fields.
func Fields(val reflect.Value, prefix string, s *internalscope.Scope) error {
	for i := range val.NumField() {
		field := val.Field(i)
		structF := val.Type().Field(i)

		// Skip unexported fields, but recurse into unexported embedded structs
		// because their exported fields are promoted and accessible.
		if !field.CanInterface() {
			if structF.Anonymous && structF.Type.Kind() == reflect.Struct {
				_, hasDefineHook := internalhooks.DefineHookRegistry[structF.Type]
				if !hasDefineHook {
					if err := Fields(field, internalpath.GetFieldName(prefix, structF), s); err != nil {
						return err
					}
				}
			}
			continue
		}

		fieldName := internalpath.GetFieldName(prefix, structF)
		// Some Go structs are handled as scalar/leaf values by built-in hooks
		// (e.g. net.IPNet), so validation must not recurse into their exported fields.
		_, hasDefineHook := internalhooks.DefineHookRegistry[structF.Type]
		isStructKind := structF.Type.Kind() == reflect.Struct && !hasDefineHook

		// Validate flagpreset tag syntax
		presets, presetErr := internaltag.ParseFlagPresets(structF.Tag.Get("flagpreset"))
		if presetErr != nil {
			return structclierrors.NewInvalidTagUsageError(fieldName, "flagpreset", presetErr.Error())
		}
		if len(presets) > 0 && isStructKind {
			return structclierrors.NewInvalidTagUsageError(fieldName, "flagpreset", "flagpreset cannot be used on struct types")
		}

		// Validate flagshort tag
		short := structF.Tag.Get("flagshort")
		if short != "" && len(short) > 1 {
			return structclierrors.NewInvalidShorthandError(fieldName, short)
		}

		// Ensure that flagshort is given to non-struct types
		if short != "" && isStructKind {
			return structclierrors.NewInvalidTagUsageError(fieldName, "flagshort", "flagshort cannot be used on struct types")
		}

		// Validate flagenv tag (can be on struct fields for inheritance)
		flagEnvValue := structF.Tag.Get("flagenv")
		flagEnvOnly := internalenv.IsEnvOnly(structF)
		if !internalenv.IsValidFlagEnvTag(flagEnvValue) {
			return structclierrors.NewInvalidFlagEnvTagError(fieldName, flagEnvValue)
		}

		// flagenv:"only" is incompatible with flag-specific tags
		if flagEnvOnly && !isStructKind {
			if short != "" {
				return structclierrors.NewConflictingTagsError(fieldName, []string{"flagenv", "flagshort"}, "flagshort cannot be used with flagenv='only'")
			}
			if len(presets) > 0 {
				return structclierrors.NewConflictingTagsError(fieldName, []string{"flagenv", "flagpreset"}, "flagpreset cannot be used with flagenv='only'")
			}
			if structF.Tag.Get("flagtype") != "" {
				return structclierrors.NewConflictingTagsError(fieldName, []string{"flagenv", "flagtype"}, "flagtype cannot be used with flagenv='only'")
			}
		}

		// Validate flagignore tag
		flagIgnoreValue, flagIgnoreErr := IsValidBoolTag(fieldName, "flagignore", structF.Tag.Get("flagignore"))
		if flagIgnoreErr != nil {
			return flagIgnoreErr
		}

		// Ensure that flagignore is given to non-struct types
		if flagIgnoreValue != nil && *flagIgnoreValue && isStructKind {
			return structclierrors.NewInvalidTagUsageError(fieldName, "flagignore", "flagignore cannot be used on struct types")
		}
		if len(presets) > 0 && flagIgnoreValue != nil && *flagIgnoreValue {
			return structclierrors.NewInvalidTagUsageError(fieldName, "flagpreset", "flagpreset cannot be used with flagignore='true'")
		}
		if flagEnvOnly && flagIgnoreValue != nil && *flagIgnoreValue {
			return structclierrors.NewConflictingTagsError(fieldName, []string{"flagenv", "flagignore"}, "mutually exclusive tags")
		}

		// Validate flagrequired tag
		flagRequiredValue, flagRequiredErr := IsValidBoolTag(fieldName, "flagrequired", structF.Tag.Get("flagrequired"))
		if flagRequiredErr != nil {
			return flagRequiredErr
		}

		// Ensure that flagrequired is given to non-struct types
		if flagRequiredValue != nil && *flagRequiredValue && isStructKind {
			return structclierrors.NewInvalidTagUsageError(fieldName, "flagrequired", "flagrequired cannot be used on struct types")
		}

		if flagRequiredValue != nil && flagIgnoreValue != nil && *flagRequiredValue && *flagIgnoreValue {
			return structclierrors.NewConflictingTagsError(fieldName, []string{"flagignore", "flagrequired"}, "mutually exclusive tags")
		}

		// Validate flaghidden tag
		flagHiddenValue, flagHiddenErr := IsValidBoolTag(fieldName, "flaghidden", structF.Tag.Get("flaghidden"))
		if flagHiddenErr != nil {
			return flagHiddenErr
		}

		// Ensure that flaghidden is given to non-struct types
		if flagHiddenValue != nil && *flagHiddenValue && isStructKind {
			return structclierrors.NewInvalidTagUsageError(fieldName, "flaghidden", "flaghidden cannot be used on struct types")
		}

		if flagHiddenValue != nil && flagIgnoreValue != nil && *flagHiddenValue && *flagIgnoreValue {
			return structclierrors.NewConflictingTagsError(fieldName, []string{"flagignore", "flaghidden"}, "mutually exclusive tags")
		}

		// NOTE: flaghidden + flagrequired is intentionally allowed.
		// Use case: flags that must be set via env var or config but should not clutter --help.

		// Check for duplicate flags
		if !isStructKind {
			// Skip ignored fields from duplicate check
			if flagIgnoreValue != nil && *flagIgnoreValue {
				continue
			}

			alias := structF.Tag.Get("flag")
			var flagName string
			if alias != "" {
				flagName = alias
			} else {
				flagName = strings.ToLower(structF.Name)
			}

			if !internaltag.IsValidFlagName(flagName) {
				return structclierrors.NewInvalidFlagNameError(fieldName, flagName)
			}

			if err := s.AddDefinedFlag(flagName, fieldName); err != nil {
				return err
			}
			for _, preset := range presets {
				if err := s.AddDefinedFlag(preset.Name, fieldName); err != nil {
					return err
				}
			}
		}

		// Recursively validate children structs
		if isStructKind {
			if err := Fields(field, fieldName, s); err != nil {
				return err
			}
		}
	}

	return nil
}


