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
	"github.com/spf13/pflag"
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

// Struct checks the coherence of definitions in the given struct
func Struct(c *cobra.Command, o any) error {
	val, err := internalreflect.GetValidValue(o)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	s := internalscope.Get(c)

	typeToFields := make(map[reflect.Type][]string)
	typeName := val.Type().Name()
	if err := Fields(val, typeName, typeToFields, s); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	for fieldType, fieldNames := range typeToFields {
		if len(fieldNames) > 1 {
			return structclierrors.NewConflictingTypeError(fieldType, fieldNames, "create distinct custom types for each field")
		}
	}

	return nil
}

// Fields recursively validates the struct fields
func Fields(val reflect.Value, prefix string, typeToFields map[reflect.Type][]string, s *internalscope.Scope) error {
	for i := range val.NumField() {
		field := val.Field(i)
		structF := val.Type().Field(i)

		// Skip unexported fields, but recurse into unexported embedded structs
		// because their exported fields are promoted and accessible.
		if !field.CanInterface() {
			if structF.Anonymous && structF.Type.Kind() == reflect.Struct {
				_, hasDefineHook := internalhooks.DefineHookRegistry[structF.Type.String()]
				if !hasDefineHook {
					if err := Fields(field, internalpath.GetFieldName(prefix, structF), typeToFields, s); err != nil {
						return err
					}
				}
			}
			continue
		}

		fieldName := internalpath.GetFieldName(prefix, structF)
		// Some Go structs are handled as scalar/leaf values by built-in hooks
		// (e.g. net.IPNet), so validation must not recurse into their exported fields.
		_, hasDefineHook := internalhooks.DefineHookRegistry[structF.Type.String()]
		isStructKind := structF.Type.Kind() == reflect.Struct && !hasDefineHook
		parts := strings.Split(fieldName, ".")
		methodFieldName := parts[len(parts)-1]

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

		// Validate flagcustom tag
		flagCustomValue, flagCustomErr := IsValidBoolTag(fieldName, "flagcustom", structF.Tag.Get("flagcustom"))
		if flagCustomErr != nil {
			return flagCustomErr
		}

		// Ensure that flagcustom is given to non-struct types
		if flagCustomValue != nil && *flagCustomValue && isStructKind {
			return structclierrors.NewInvalidTagUsageError(fieldName, "flagcustom", "flagcustom cannot be used on struct types")
		}

		// Reject flagcustom + flagenv:"only" before validating hooks (avoids confusing hook errors)
		if flagCustomValue != nil && *flagCustomValue && internalenv.IsEnvOnly(structF) && !isStructKind {
			return structclierrors.NewConflictingTagsError(fieldName, []string{"flagenv", "flagcustom"}, "flagcustom cannot be used with flagenv='only'")
		}

		// Validate the define and decode hooks when flagcustom is true
		if flagCustomValue != nil && *flagCustomValue && !isStructKind {
			// Map current field name to its custom type
			if !internaltag.IsStandardType(structF.Type) {
				typeToFields[structF.Type] = append(typeToFields[structF.Type], fieldName)
			}
			if err := validateCustomFlag(val, methodFieldName, structF.Type.String()); err != nil {
				return err
			}
		}

		// Validate flagenv tag (can be on struct fields for inheritance)
		flagEnvValue := structF.Tag.Get("flagenv")
		flagEnvOnly := internalenv.IsEnvOnly(structF)
		if !internalenv.IsValidFlagEnvTag(flagEnvValue) {
			return structclierrors.NewInvalidBooleanTagError(fieldName, "flagenv", flagEnvValue)
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
			if flagCustomValue != nil && *flagCustomValue {
				return structclierrors.NewConflictingTagsError(fieldName, []string{"flagenv", "flagcustom"}, "flagcustom cannot be used with flagenv='only'")
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
		if !isStructKind && !(flagIgnoreValue != nil && *flagIgnoreValue) {
			if err := validateCompletionHook(val, methodFieldName); err != nil {
				return err
			}
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
			if err := Fields(field, fieldName, typeToFields, s); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateDefineHookSignature(m reflect.Value) error {
	expectedType := reflect.TypeOf((*internalhooks.DefineHookFunc)(nil)).Elem()
	actualType := m.Type()

	if actualType.NumIn() != expectedType.NumIn() || actualType.NumOut() != expectedType.NumOut() {
		var fx internalhooks.DefineHookFunc

		return fmt.Errorf("define hook must have signature: %s", internalreflect.Signature(fx))
	}

	// Check input types
	for i := range actualType.NumIn() {
		if actualType.In(i) != expectedType.In(i) {
			return fmt.Errorf("define hook parameter %d has wrong type: expected %v, got %v", i, expectedType.In(i), actualType.In(i))
		}
	}

	// Check return types
	pflagValueType := reflect.TypeOf((*pflag.Value)(nil)).Elem()
	if !actualType.Out(0).Implements(pflagValueType) {
		return fmt.Errorf("define hook first return value must be a pflag.Value")
	}
	if actualType.Out(1).Kind() != reflect.String {
		return fmt.Errorf("define hook second return value must be a string")
	}

	return nil
}

func validateDecodeHookSignature(m reflect.Value) error {
	expectedType := reflect.TypeOf((*internalhooks.DecodeHookFunc)(nil)).Elem()
	actualType := m.Type()

	if actualType.NumIn() != expectedType.NumIn() || actualType.NumOut() != expectedType.NumOut() {
		var fx internalhooks.DecodeHookFunc

		return fmt.Errorf("decode hook must have signature: %s", internalreflect.Signature(fx))
	}

	if actualType.In(0) != expectedType.In(0) {
		return fmt.Errorf("decode hook input parameter has wrong type: expected %v, got %v", expectedType.In(0), actualType.In(0))
	}

	if actualType.Out(0) != expectedType.Out(0) ||
		actualType.Out(1) != expectedType.Out(1) {
		return fmt.Errorf("decode hook must return (any, error)")
	}

	return nil
}

func validateCompleteHookSignature(m reflect.Value) error {
	expectedType := reflect.TypeOf((*internalhooks.CompleteHookFunc)(nil)).Elem()
	actualType := m.Type()

	if actualType.NumIn() != expectedType.NumIn() || actualType.NumOut() != expectedType.NumOut() {
		var fx internalhooks.CompleteHookFunc

		return fmt.Errorf("complete hook must have signature: %s", internalreflect.Signature(fx))
	}

	for i := range actualType.NumIn() {
		if actualType.In(i) != expectedType.In(i) {
			return fmt.Errorf("complete hook parameter %d has wrong type: expected %v, got %v", i, expectedType.In(i), actualType.In(i))
		}
	}

	for i := range actualType.NumOut() {
		if actualType.Out(i) != expectedType.Out(i) {
			return fmt.Errorf("complete hook return value %d has wrong type: expected %v, got %v", i, expectedType.Out(i), actualType.Out(i))
		}
	}

	return nil
}

// validateCustomFlag validates that a custom flag has proper define and decode mechanisms
func validateCustomFlag(structValue reflect.Value, fieldName, fieldType string) error {
	// Get pointer to struct to access methods
	structPtr := internalreflect.GetStructPtr(structValue)
	if !structPtr.IsValid() {
		return fmt.Errorf("cannot get pointer to struct for field '%s'", fieldName)
	}

	// Check if struct has Define<FieldName> method
	defineMethodName := fmt.Sprintf("Define%s", fieldName)
	defineHookFunc := structPtr.MethodByName(defineMethodName)

	// Check if struct has Decode<FieldName> method
	decodeMethodName := fmt.Sprintf("Decode%s", fieldName)
	decodeHookFunc := structPtr.MethodByName(decodeMethodName)

	// Case 1: User has defined custom methods
	if defineHookFunc.IsValid() {
		// Must have corresponding decode method
		if !decodeHookFunc.IsValid() {
			return structclierrors.NewMissingDecodeHookError(fieldName, decodeMethodName)
		}

		// Validate signatures
		if err := validateDefineHookSignature(defineHookFunc); err != nil {
			return structclierrors.NewInvalidDefineHookSignatureError(fieldName, defineMethodName, err)
		}
		if err := validateDecodeHookSignature(decodeHookFunc); err != nil {
			return structclierrors.NewInvalidDecodeHookSignatureError(fieldName, decodeMethodName, err)
		}

		return nil
	}

	// Check registries
	_, inDefineRegistry := internalhooks.DefineHookRegistry[fieldType]
	_, inDecodeRegistry := internalhooks.DecodeHookRegistry[fieldType]

	// Case 2: Check registry
	if inDefineRegistry {
		if !inDecodeRegistry {
			return fmt.Errorf("internal error: missing decode hook for built-in type %s", fieldType)
		}

		return nil
	}

	// Case 3: No define mechanism found
	return structclierrors.NewMissingDefineHookError(fieldName, defineMethodName)
}

func validateCompletionHook(structValue reflect.Value, fieldName string) error {
	structPtr := internalreflect.GetStructPtr(structValue)
	if !structPtr.IsValid() {
		return fmt.Errorf("cannot get pointer to struct for field '%s'", fieldName)
	}

	methodName := fmt.Sprintf("Complete%s", fieldName)
	completeHookFunc := structPtr.MethodByName(methodName)
	if !completeHookFunc.IsValid() {
		return nil
	}

	if err := validateCompleteHookSignature(completeHookFunc); err != nil {
		return structclierrors.NewInvalidCompleteHookSignatureError(fieldName, methodName, err)
	}

	return nil
}
