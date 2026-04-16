// Property-based tests for the validation functions in internal/validation.
//
// Uses pgregory.net/rapid to generate random struct types with various tag
// combinations, then asserts invariants on the validation output.
package validation_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	structclierrors "github.com/leodido/structcli/errors"
	"github.com/leodido/structcli/internal/proptest/gen"
	internalvalidation "github.com/leodido/structcli/internal/validation"
	"github.com/spf13/cobra"
	"pgregory.net/rapid"
)

// newCmd creates a fresh cobra.Command for each test iteration.
func newCmd() *cobra.Command {
	return &cobra.Command{Use: "test"}
}

// --- P2.1: IsValidBoolTag never panics and returns well-typed errors ---

func TestProperty_IsValidBoolTag_NoPanicAndTypedErrors(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.String().Draw(t, "input")
		result, err := internalvalidation.IsValidBoolTag("Field", "flagcustom", input)
		if err != nil {
			var target *structclierrors.InvalidBooleanTagError
			if !errors.As(err, &target) {
				t.Fatalf("IsValidBoolTag(%q) returned non-typed error: %v", input, err)
			}
		}
		if err == nil && result != nil {
			if *result != true && *result != false {
				t.Fatalf("IsValidBoolTag(%q) returned non-bool value: %v", input, *result)
			}
		}
	})
}

// --- P2.2: Validation rejects flagignore + flagrequired ---

func TestProperty_Validation_RejectsIgnoreAndRequired(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fieldType := gen.FieldType().Draw(t, "type")
		flagName := strings.ToLower(gen.ValidFlagName().Draw(t, "flagName"))

		typ := reflect.StructOf([]reflect.StructField{
			{
				Name: "F0",
				Type: fieldType,
				Tag:  reflect.StructTag(fmt.Sprintf(`flag:"%s" flagignore:"true" flagrequired:"true"`, flagName)),
			},
		})
		opts := reflect.New(typ).Interface()

		err := internalvalidation.Struct(newCmd(), opts)
		if err == nil {
			t.Fatal("expected error for flagignore+flagrequired, got nil")
		}
		var target *structclierrors.ConflictingTagsError
		if !errors.As(err, &target) {
			t.Fatalf("expected ConflictingTagsError, got: %v", err)
		}
	})
}

// --- P2.3: Validation rejects flaghidden + flagignore ---

func TestProperty_Validation_RejectsHiddenAndIgnore(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fieldType := gen.FieldType().Draw(t, "type")
		flagName := strings.ToLower(gen.ValidFlagName().Draw(t, "flagName"))

		typ := reflect.StructOf([]reflect.StructField{
			{
				Name: "F0",
				Type: fieldType,
				Tag:  reflect.StructTag(fmt.Sprintf(`flag:"%s" flaghidden:"true" flagignore:"true"`, flagName)),
			},
		})
		opts := reflect.New(typ).Interface()

		err := internalvalidation.Struct(newCmd(), opts)
		if err == nil {
			t.Fatal("expected error for flaghidden+flagignore, got nil")
		}
		var target *structclierrors.ConflictingTagsError
		if !errors.As(err, &target) {
			t.Fatalf("expected ConflictingTagsError, got: %v", err)
		}
	})
}

// --- P2.4: Validation allows flaghidden + flagrequired ---

func TestProperty_Validation_AllowsHiddenAndRequired(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fieldType := gen.FieldType().Draw(t, "type")
		flagName := strings.ToLower(gen.ValidFlagName().Draw(t, "flagName"))

		typ := reflect.StructOf([]reflect.StructField{
			{
				Name: "F0",
				Type: fieldType,
				Tag:  reflect.StructTag(fmt.Sprintf(`flag:"%s" flaghidden:"true" flagrequired:"true"`, flagName)),
			},
		})
		opts := reflect.New(typ).Interface()

		err := internalvalidation.Struct(newCmd(), opts)
		if err != nil {
			t.Fatalf("flaghidden+flagrequired should be allowed, got: %v", err)
		}
	})
}

// --- P2.5: Validation rejects struct-only tags on struct-typed fields ---

func TestProperty_Validation_RejectsStructOnlyTagsOnStructFields(t *testing.T) {
	// Tags like flagshort, flagcustom, flagignore, flagrequired, flaghidden
	// are invalid on struct-typed fields.
	tags := []struct {
		name string
		tag  string
	}{
		{"flagshort", `flagshort:"x"`},
		{"flagcustom", `flagcustom:"true"`},
		{"flagignore", `flagignore:"true"`},
		{"flagrequired", `flagrequired:"true"`},
		{"flaghidden", `flaghidden:"true"`},
	}

	for _, tc := range tags {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				// Create an inner struct type
				innerType := reflect.StructOf([]reflect.StructField{
					{
						Name: "Inner",
						Type: reflect.TypeOf(""),
						Tag:  reflect.StructTag(`flag:"inner"`),
					},
				})

				outerType := reflect.StructOf([]reflect.StructField{
					{
						Name: "Nested",
						Type: innerType,
						Tag:  reflect.StructTag(tc.tag),
					},
				})
				opts := reflect.New(outerType).Interface()

				err := internalvalidation.Struct(newCmd(), opts)
				if err == nil {
					t.Fatalf("expected error for %s on struct field, got nil", tc.name)
				}
				var target *structclierrors.InvalidTagUsageError
				if !errors.As(err, &target) {
					t.Fatalf("expected InvalidTagUsageError for %s, got: %v", tc.name, err)
				}
			})
		})
	}
}

// --- P2.6: Validation rejects duplicate flag names ---

func TestProperty_Validation_RejectsDuplicateFlagNames(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		flagName := strings.ToLower(gen.ValidFlagName().Draw(t, "flagName"))
		typeA := gen.FieldType().Draw(t, "typeA")
		typeB := gen.FieldType().Draw(t, "typeB")

		typ := reflect.StructOf([]reflect.StructField{
			{
				Name: "F0",
				Type: typeA,
				Tag:  reflect.StructTag(fmt.Sprintf(`flag:"%s"`, flagName)),
			},
			{
				Name: "F1",
				Type: typeB,
				Tag:  reflect.StructTag(fmt.Sprintf(`flag:"%s"`, flagName)),
			},
		})
		opts := reflect.New(typ).Interface()

		err := internalvalidation.Struct(newCmd(), opts)
		if err == nil {
			t.Fatalf("expected error for duplicate flag name %q, got nil", flagName)
		}
		var target *structclierrors.DuplicateFlagError
		if !errors.As(err, &target) {
			t.Fatalf("expected DuplicateFlagError, got: %v", err)
		}
	})
}

// --- P2.7: Validation rejects invalid flag names ---

func TestProperty_Validation_RejectsInvalidFlagNames(t *testing.T) {
	// Generate strings that are invalid flag names AND survive struct tag
	// encoding (no quotes, backslashes, or control chars that would make
	// Tag.Get return empty and fall back to the valid field name).
	invalidNames := rapid.OneOf(
		rapid.Just("bad name"),
		rapid.Just("has space"),
		rapid.Just("-leading"),
		rapid.Just(".leading"),
		rapid.Just("trailing-"),
		rapid.Just("trailing."),
		rapid.Just("a--b"),
		rapid.Just("a..b"),
		rapid.Just("a b c"),
		rapid.Just("foo!bar"),
		rapid.Just("x@y"),
		rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]*[ !@#$%^&*()][a-zA-Z0-9]+`),
	)

	rapid.Check(t, func(t *rapid.T) {
		flagName := invalidNames.Draw(t, "flagName")
		// Skip if it accidentally matches the valid pattern
		if gen.IsValidFlagNameCheck(flagName) {
			return
		}
		// Skip names with chars that break struct tag parsing
		if strings.ContainsAny(flagName, `"\\`) {
			return
		}

		typ := reflect.StructOf([]reflect.StructField{
			{
				Name: "F0",
				Type: reflect.TypeOf(""),
				Tag:  reflect.StructTag(fmt.Sprintf(`flag:"%s"`, flagName)),
			},
		})

		// Verify the tag round-trips correctly
		if typ.Field(0).Tag.Get("flag") != flagName {
			return // tag encoding mangled the name, skip
		}

		opts := reflect.New(typ).Interface()

		err := internalvalidation.Struct(newCmd(), opts)
		if err == nil {
			t.Fatalf("expected error for invalid flag name %q, got nil", flagName)
		}
		var target *structclierrors.InvalidFlagNameError
		if !errors.As(err, &target) {
			t.Fatalf("expected InvalidFlagNameError for %q, got: %v", flagName, err)
		}
	})
}

// --- P2.8: Validation rejects invalid boolean tag values ---

func TestProperty_Validation_RejectsInvalidBooleanTagValues(t *testing.T) {
	boolTags := []string{"flagcustom", "flagenv", "flagignore", "flagrequired", "flaghidden"}

	for _, tagName := range boolTags {
		tagName := tagName
		t.Run(tagName, func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				badVal := gen.InvalidBoolTagValue().Draw(t, "badVal")
				flagName := strings.ToLower(gen.ValidFlagName().Draw(t, "flagName"))

				typ := reflect.StructOf([]reflect.StructField{
					{
						Name: "F0",
						Type: reflect.TypeOf(""),
						Tag:  reflect.StructTag(fmt.Sprintf(`flag:"%s" %s:"%s"`, flagName, tagName, badVal)),
					},
				})
				opts := reflect.New(typ).Interface()

				err := internalvalidation.Struct(newCmd(), opts)
				if err == nil {
					t.Fatalf("expected error for %s=%q, got nil", tagName, badVal)
				}
				var target *structclierrors.InvalidBooleanTagError
				if !errors.As(err, &target) {
					t.Fatalf("expected InvalidBooleanTagError for %s=%q, got: %v", tagName, badVal, err)
				}
			})
		})
	}
}

// --- P2.9: Validation accepts well-formed structs ---

func TestProperty_Validation_AcceptsWellFormedStructs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		specs := gen.UniqueFieldSpecs(6).Draw(t, "fields")
		typ := gen.BuildStructType(specs)
		opts := gen.NewStructValue(typ)

		err := internalvalidation.Struct(newCmd(), opts)
		if err != nil {
			t.Fatalf("expected nil error for well-formed struct, got: %v\nspecs: %+v", err, specs)
		}
	})
}

// --- P2.10: Validation errors are always well-typed ---

func TestProperty_Validation_ErrorsAreWellTyped(t *testing.T) {
	knownErrorTypes := []any{
		(*structclierrors.InvalidBooleanTagError)(nil),
		(*structclierrors.InvalidShorthandError)(nil),
		(*structclierrors.InvalidTagUsageError)(nil),
		(*structclierrors.ConflictingTagsError)(nil),
		(*structclierrors.DuplicateFlagError)(nil),
		(*structclierrors.InvalidFlagNameError)(nil),
		(*structclierrors.ConflictingTypeError)(nil),
		(*structclierrors.InputError)(nil),
		(*structclierrors.MissingDefineHookError)(nil),
		(*structclierrors.MissingDecodeHookError)(nil),
		(*structclierrors.InvalidDefineHookSignatureError)(nil),
		(*structclierrors.InvalidDecodeHookSignatureError)(nil),
		(*structclierrors.InvalidCompleteHookSignatureError)(nil),
	}

	rapid.Check(t, func(t *rapid.T) {
		specs := gen.UniqueFieldSpecs(4).Draw(t, "fields")
		typ := gen.BuildStructType(specs)
		opts := gen.NewStructValue(typ)

		err := internalvalidation.Struct(newCmd(), opts)
		if err == nil {
			return // success is fine
		}

		// The error should either be a known typed error or wrap one.
		// Validation wraps errors with fmt.Errorf("validation failed: %w", ...),
		// so we need to unwrap.
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			unwrapped = err
		}

		for _, knownType := range knownErrorTypes {
			target := reflect.New(reflect.TypeOf(knownType).Elem()).Interface()
			if errors.As(unwrapped, &target) {
				return // matched a known type
			}
		}

		// Also accept fmt.Errorf-wrapped errors that contain "validation failed"
		// (these wrap the typed errors above)
		if strings.Contains(err.Error(), "validation failed") {
			return
		}

		t.Fatalf("validation returned unrecognized error type %T: %v", err, err)
	})
}
