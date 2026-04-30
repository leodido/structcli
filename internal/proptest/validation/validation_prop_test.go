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

// --- P2.5: Validation rejects leaf-only tags on struct-typed fields ---

func TestProperty_Validation_RejectsLeafOnlyTagsOnStructFields(t *testing.T) {
	// Tags like flagshort, flagcustom, flagignore, flagrequired, flaghidden
	// are valid only on leaf (non-struct) fields and rejected on struct fields.
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
	// flagenv is excluded: it accepts "only" in addition to booleans and uses InvalidFlagEnvTagError.
	boolTags := []string{"flagcustom", "flagignore", "flagrequired", "flaghidden"}

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

// --- P2.8b: flagenv rejects invalid values with InvalidFlagEnvTagError ---

func TestProperty_Validation_RejectsInvalidFlagEnvValues(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		badVal := gen.InvalidBoolTagValue().Draw(t, "badVal")
		flagName := strings.ToLower(gen.ValidFlagName().Draw(t, "flagName"))

		typ := reflect.StructOf([]reflect.StructField{
			{
				Name: "F0",
				Type: reflect.TypeOf(""),
				Tag:  reflect.StructTag(fmt.Sprintf(`flag:"%s" flagenv:"%s"`, flagName, badVal)),
			},
		})
		opts := reflect.New(typ).Interface()

		err := internalvalidation.Struct(newCmd(), opts)
		if err == nil {
			t.Fatalf("expected error for flagenv=%q, got nil", badVal)
		}
		var target *structclierrors.InvalidFlagEnvTagError
		if !errors.As(err, &target) {
			t.Fatalf("expected InvalidFlagEnvTagError for flagenv=%q, got: %v", badVal, err)
		}
	})
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

// --- P2.9b: flagenv:"only" rejects incompatible flag-specific tags ---

func TestProperty_Validation_EnvOnlyRejectsIncompatibleTags(t *testing.T) {
	// flagcustom is omitted: it requires matching Define/Decode hook methods
	// on the struct, which cannot be generated via reflect.StructOf.
	incompatibleTags := []struct {
		tagName string
		tagVal  string
	}{
		{"flagshort", "x"},
		{"flagpreset", "max=10"},
		{"flagtype", "count"},
	}

	for _, tc := range incompatibleTags {
		tc := tc
		t.Run(tc.tagName, func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				flagName := strings.ToLower(gen.ValidFlagName().Draw(t, "flagName"))

				typ := reflect.StructOf([]reflect.StructField{
					{
						Name: "F0",
						Type: reflect.TypeOf(""),
						Tag:  reflect.StructTag(fmt.Sprintf(`flag:"%s" flagenv:"only" %s:"%s"`, flagName, tc.tagName, tc.tagVal)),
					},
				})
				opts := reflect.New(typ).Interface()

				err := internalvalidation.Struct(newCmd(), opts)
				if err == nil {
					t.Fatalf("expected error for flagenv='only' + %s=%q, got nil", tc.tagName, tc.tagVal)
				}
				var target *structclierrors.ConflictingTagsError
				if !errors.As(err, &target) {
					t.Fatalf("expected ConflictingTagsError for flagenv='only' + %s, got: %v", tc.tagName, err)
				}
			})
		})
	}
}

// --- P2.9c: flagenv:"only" is accepted on well-formed fields ---

func TestProperty_Validation_EnvOnlyAcceptedAlone(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		flagName := strings.ToLower(gen.ValidFlagName().Draw(t, "flagName"))

		typ := reflect.StructOf([]reflect.StructField{
			{
				Name: "F0",
				Type: reflect.TypeOf(""),
				Tag:  reflect.StructTag(fmt.Sprintf(`flag:"%s" flagenv:"only" flagdescr:"test"`, flagName)),
			},
		})
		opts := reflect.New(typ).Interface()

		err := internalvalidation.Struct(newCmd(), opts)
		if err != nil {
			t.Fatalf("expected nil error for flagenv='only' with valid flag name %q, got: %v", flagName, err)
		}
	})
}

// --- P2.10: Validation errors are always well-typed ---

func TestProperty_Validation_ErrorsAreWellTyped(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		specs := gen.UniqueFieldSpecs(4).Draw(t, "fields")
		typ := gen.BuildStructType(specs)
		opts := gen.NewStructValue(typ)

		err := internalvalidation.Struct(newCmd(), opts)
		if err == nil {
			return // success is fine
		}

		// Validation wraps errors with fmt.Errorf("validation failed: %w", ...),
		// so unwrap first.
		inner := errors.Unwrap(err)
		if inner == nil {
			inner = err
		}

		// Check against each known error type with concrete typed pointers.
		// Using *any as the errors.As target would match everything; each
		// check must use a concretely-typed pointer variable.
		var (
			invalidBoolTag     *structclierrors.InvalidBooleanTagError
			invalidShorthand   *structclierrors.InvalidShorthandError
			invalidTagUsage    *structclierrors.InvalidTagUsageError
			conflictingTags    *structclierrors.ConflictingTagsError
			duplicateFlag      *structclierrors.DuplicateFlagError
			invalidFlagName    *structclierrors.InvalidFlagNameError
			conflictingType    *structclierrors.ConflictingTypeError
			inputErr           *structclierrors.InputError
			missingDefineHook  *structclierrors.MissingDefineHookError
			missingDecodeHook  *structclierrors.MissingDecodeHookError
			invalidDefineSig   *structclierrors.InvalidDefineHookSignatureError
			invalidDecodeSig   *structclierrors.InvalidDecodeHookSignatureError
			invalidCompleteSig *structclierrors.InvalidCompleteHookSignatureError
		)

		switch {
		case errors.As(inner, &invalidBoolTag):
		case errors.As(inner, &invalidShorthand):
		case errors.As(inner, &invalidTagUsage):
		case errors.As(inner, &conflictingTags):
		case errors.As(inner, &duplicateFlag):
		case errors.As(inner, &invalidFlagName):
		case errors.As(inner, &conflictingType):
		case errors.As(inner, &inputErr):
		case errors.As(inner, &missingDefineHook):
		case errors.As(inner, &missingDecodeHook):
		case errors.As(inner, &invalidDefineSig):
		case errors.As(inner, &invalidDecodeSig):
		case errors.As(inner, &invalidCompleteSig):
		default:
			t.Fatalf("validation returned unrecognized error type %T: %v", inner, inner)
		}
	})
}
