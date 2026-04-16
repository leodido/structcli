// Package gen provides shared rapid generators for property-based tests.
//
// Generators produce random struct types, tag sets, and field types
// for use in validation and Define() property tests.
package gen

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"pgregory.net/rapid"
)

var validFlagNameRegex = regexp.MustCompile(`^[a-zA-Z0-9]+([.-][a-zA-Z0-9]+)*$`)

// Supported field types for generation.
var SupportedFieldTypes = []reflect.Type{
	reflect.TypeOf(false),          // bool
	reflect.TypeOf(""),             // string
	reflect.TypeOf(int(0)),         // int
	reflect.TypeOf(int8(0)),        // int8
	reflect.TypeOf(int16(0)),       // int16
	reflect.TypeOf(int32(0)),       // int32
	reflect.TypeOf(int64(0)),       // int64
	reflect.TypeOf(uint(0)),        // uint
	reflect.TypeOf(uint8(0)),       // uint8
	reflect.TypeOf(uint16(0)),      // uint16
	reflect.TypeOf(uint32(0)),      // uint32
	reflect.TypeOf(uint64(0)),      // uint64
	reflect.TypeOf(float32(0)),     // float32
	reflect.TypeOf(float64(0)),     // float64
	reflect.TypeOf([]string(nil)),  // []string
	reflect.TypeOf([]int(nil)),     // []int
}

// FieldType draws a random supported field type.
func FieldType() *rapid.Generator[reflect.Type] {
	return rapid.SampledFrom(SupportedFieldTypes)
}

// ValidFlagName produces strings matching ^[a-zA-Z0-9]+([.-][a-zA-Z0-9]+)*$.
func ValidFlagName() *rapid.Generator[string] {
	segment := rapid.StringMatching(`[a-zA-Z0-9]+`)
	sep := rapid.SampledFrom([]string{".", "-"})

	return rapid.Custom(func(t *rapid.T) string {
		n := rapid.IntRange(1, 3).Draw(t, "segments")
		parts := make([]string, n)
		for i := range n {
			parts[i] = segment.Draw(t, "segment")
		}
		if n == 1 {
			return parts[0]
		}
		var b strings.Builder
		b.WriteString(parts[0])
		for i := 1; i < n; i++ {
			b.WriteString(sep.Draw(t, "sep"))
			b.WriteString(parts[i])
		}
		return b.String()
	})
}

// BoolTagValue draws a value suitable for boolean struct tags.
func BoolTagValue() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"true", "false", "1", "0", "TRUE", "FALSE"})
}

// InvalidBoolTagValue draws a string that is NOT a valid boolean.
func InvalidBoolTagValue() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"yes", "no", "2", "maybe", "on", "off", "TRUE!", "abc"})
}

// TagSet represents a set of struct tags for a single field.
type TagSet struct {
	Flag         string // flag alias (may be empty)
	FlagShort    string // single char shorthand
	FlagDescr    string
	FlagGroup    string
	FlagHidden   string // "true" or "false" or ""
	FlagRequired string // "true" or "false" or ""
	FlagIgnore   string // "true" or "false" or ""
	FlagCustom   string // "true" or "false" or ""
	FlagEnv      string // "true" or "false" or ""
	FlagPreset   string
	Default      string
}

// ToStructTag converts a TagSet to a reflect.StructTag string.
func (ts TagSet) ToStructTag() reflect.StructTag {
	var parts []string
	add := func(key, val string) {
		if val != "" {
			parts = append(parts, fmt.Sprintf(`%s:"%s"`, key, val))
		}
	}
	add("flag", ts.Flag)
	add("flagshort", ts.FlagShort)
	add("flagdescr", ts.FlagDescr)
	add("flaggroup", ts.FlagGroup)
	add("flaghidden", ts.FlagHidden)
	add("flagrequired", ts.FlagRequired)
	add("flagignore", ts.FlagIgnore)
	add("flagcustom", ts.FlagCustom)
	add("flagenv", ts.FlagEnv)
	add("flagpreset", ts.FlagPreset)
	add("default", ts.Default)
	return reflect.StructTag(strings.Join(parts, " "))
}

// ValidTagSet generates a well-formed tag set with no conflicts.
// The flagName parameter is used as the flag alias if non-empty.
func ValidTagSet(flagName string) *rapid.Generator[TagSet] {
	return rapid.Custom(func(t *rapid.T) TagSet {
		ts := TagSet{}
		if flagName != "" {
			ts.Flag = flagName
		}

		// Optionally add a shorthand (single ASCII letter)
		if rapid.Bool().Draw(t, "hasShort") {
			ts.FlagShort = string(rapid.ByteRange('a', 'z').Draw(t, "short"))
		}

		// Optionally add a description
		if rapid.Bool().Draw(t, "hasDescr") {
			ts.FlagDescr = rapid.StringMatching(`[a-zA-Z0-9 ]{1,30}`).Draw(t, "descr")
		}

		// Optionally add a group
		if rapid.Bool().Draw(t, "hasGroup") {
			ts.FlagGroup = rapid.StringMatching(`[A-Z][a-zA-Z]{2,10}`).Draw(t, "group")
		}

		// Hidden and required can coexist, but neither can coexist with ignore
		useIgnore := rapid.Bool().Draw(t, "useIgnore")
		if useIgnore {
			ts.FlagIgnore = "true"
			// No hidden, required, or preset when ignored
		} else {
			if rapid.Bool().Draw(t, "hidden") {
				ts.FlagHidden = "true"
			}
			if rapid.Bool().Draw(t, "required") {
				ts.FlagRequired = "true"
			}
		}

		return ts
	})
}

// ArbitraryTagSet generates a tag set with arbitrary (possibly invalid) values.
func ArbitraryTagSet() *rapid.Generator[TagSet] {
	return rapid.Custom(func(t *rapid.T) TagSet {
		ts := TagSet{}
		if rapid.Bool().Draw(t, "hasFlag") {
			ts.Flag = rapid.String().Draw(t, "flag")
		}
		if rapid.Bool().Draw(t, "hasShort") {
			ts.FlagShort = rapid.String().Draw(t, "short")
		}
		if rapid.Bool().Draw(t, "hasHidden") {
			ts.FlagHidden = rapid.String().Draw(t, "hidden")
		}
		if rapid.Bool().Draw(t, "hasRequired") {
			ts.FlagRequired = rapid.String().Draw(t, "required")
		}
		if rapid.Bool().Draw(t, "hasIgnore") {
			ts.FlagIgnore = rapid.String().Draw(t, "ignore")
		}
		if rapid.Bool().Draw(t, "hasCustom") {
			ts.FlagCustom = rapid.String().Draw(t, "custom")
		}
		if rapid.Bool().Draw(t, "hasEnv") {
			ts.FlagEnv = rapid.String().Draw(t, "env")
		}
		return ts
	})
}

// FieldSpec describes a generated struct field.
type FieldSpec struct {
	Name string
	Type reflect.Type
	Tags TagSet
}

// UniqueFieldSpecs generates 1–maxFields field specs with unique valid flag names.
func UniqueFieldSpecs(maxFields int) *rapid.Generator[[]FieldSpec] {
	return rapid.Custom(func(t *rapid.T) []FieldSpec {
		n := rapid.IntRange(1, maxFields).Draw(t, "fieldCount")
		usedNames := make(map[string]struct{})
		fields := make([]FieldSpec, 0, n)

		for len(fields) < n {
			flagName := strings.ToLower(ValidFlagName().Draw(t, "flagName"))
			if _, dup := usedNames[flagName]; dup {
				continue
			}
			usedNames[flagName] = struct{}{}

			fieldType := FieldType().Draw(t, "fieldType")
			tags := ValidTagSet(flagName).Draw(t, "tags")

			// Field name must be exported (uppercase first letter)
			exportedName := fmt.Sprintf("F%d", len(fields))

			fields = append(fields, FieldSpec{
				Name: exportedName,
				Type: fieldType,
				Tags: tags,
			})
		}
		return fields
	})
}

// BuildStructType creates a reflect.Type from field specs.
func BuildStructType(specs []FieldSpec) reflect.Type {
	fields := make([]reflect.StructField, len(specs))
	for i, s := range specs {
		fields[i] = reflect.StructField{
			Name: s.Name,
			Type: s.Type,
			Tag:  s.Tags.ToStructTag(),
		}
	}
	return reflect.StructOf(fields)
}

// NewStructValue creates a zero-valued pointer to a struct of the given type.
func NewStructValue(typ reflect.Type) any {
	return reflect.New(typ).Interface()
}

// IsValidFlagNameCheck checks if a string matches the valid flag name regex.
// Duplicated here to avoid importing internal/tag from the gen package.
func IsValidFlagNameCheck(name string) bool {
	return validFlagNameRegex.MatchString(name)
}

// FlagNameForField returns the flag name that Define/validation would use.
func FlagNameForField(spec FieldSpec) string {
	if spec.Tags.Flag != "" {
		return spec.Tags.Flag
	}
	return strings.ToLower(spec.Name)
}
