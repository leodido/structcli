// Property-based tests for the tag parsing functions in internal/tag.
//
// Uses pgregory.net/rapid to generate random tag values and flag names,
// then asserts invariants that must hold for all inputs.
package tag_test

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	internaltag "github.com/leodido/structcli/internal/tag"
	"pgregory.net/rapid"
)

var validFlagNameRegex = regexp.MustCompile(`^[a-zA-Z0-9]+([.-][a-zA-Z0-9]+)*$`)

// --- Generators ---

// genValidFlagName produces strings that match the flag name regex.
func genValidFlagName() *rapid.Generator[string] {
	// A segment is 1+ alphanumeric chars.
	segment := rapid.StringMatching(`[a-zA-Z0-9]+`)
	// A separator is either '.' or '-'.
	sep := rapid.SampledFrom([]string{".", "-"})

	return rapid.Custom(func(t *rapid.T) string {
		n := rapid.IntRange(1, 4).Draw(t, "segments")
		parts := make([]string, n)
		for i := range n {
			parts[i] = segment.Draw(t, "segment")
		}
		if n == 1 {
			return parts[0]
		}
		// Join with random separators
		var b strings.Builder
		b.WriteString(parts[0])
		for i := 1; i < n; i++ {
			b.WriteString(sep.Draw(t, "sep"))
			b.WriteString(parts[i])
		}
		return b.String()
	})
}

// genArbitraryString produces arbitrary strings including edge cases.
func genArbitraryString() *rapid.Generator[string] {
	return rapid.OneOf(
		rapid.String(),
		rapid.Just(""),
		rapid.Just(" "),
		rapid.Just("\t"),
		rapid.Just("\x00"),
		rapid.StringMatching(`[^a-zA-Z0-9._-]+`),
	)
}

// genPresetEntry produces a single "name=value" preset entry with a valid name.
func genPresetEntry() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		name := genValidFlagName().Draw(t, "name")
		value := rapid.String().Draw(t, "value")
		return name + "=" + value
	})
}

// genValidPresetTag produces a well-formed preset tag string with N entries.
func genValidPresetTag() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		n := rapid.IntRange(1, 5).Draw(t, "count")
		// Generate unique names
		seen := make(map[string]struct{})
		entries := make([]string, 0, n)
		for len(entries) < n {
			name := genValidFlagName().Draw(t, "name")
			lower := strings.ToLower(name)
			if _, dup := seen[lower]; dup {
				continue
			}
			seen[lower] = struct{}{}
			value := rapid.String().Draw(t, "value")
			// Ensure value doesn't contain the separator we'll use
			entries = append(entries, name+"="+value)
		}
		// Use semicolon as separator (the dominant one)
		return strings.Join(entries, ";")
	})
}

// genArbitraryPresetTag produces arbitrary strings that may or may not be valid preset tags.
func genArbitraryPresetTag() *rapid.Generator[string] {
	return rapid.OneOf(
		rapid.String(),
		rapid.Just(""),
		rapid.Just("="),
		rapid.Just(";"),
		rapid.Just(","),
		rapid.Just(";;"),
		rapid.Just("a=1;;b=2"),
		rapid.Just("=value"),
		rapid.StringMatching(`[a-zA-Z0-9=;,. -]+`),
	)
}

// genBoolTagValue produces strings that may or may not be valid boolean tag values.
func genBoolTagValue() *rapid.Generator[string] {
	return rapid.OneOf(
		rapid.SampledFrom([]string{"true", "false", "1", "0", "TRUE", "FALSE", "True", "False", "t", "f", "T", "F"}),
		rapid.String(),
		rapid.Just(""),
		rapid.Just("yes"),
		rapid.Just("no"),
		rapid.Just("2"),
		rapid.Just("maybe"),
	)
}

// --- Properties ---

// P1.1: IsValidFlagName is consistent with the regex.
func TestProperty_IsValidFlagName_ConsistentWithRegex(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.String().Draw(t, "input")
		got := internaltag.IsValidFlagName(s)
		want := validFlagNameRegex.MatchString(s)
		if got != want {
			t.Fatalf("IsValidFlagName(%q) = %v, regex says %v", s, got, want)
		}
	})
}

// P1.1b: Valid flag names always pass.
func TestProperty_IsValidFlagName_ValidNamesPass(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := genValidFlagName().Draw(t, "name")
		if !internaltag.IsValidFlagName(name) {
			t.Fatalf("IsValidFlagName(%q) = false, expected true", name)
		}
	})
}

// P1.2: ParseFlagPresets round-trip consistency for valid inputs.
func TestProperty_ParseFlagPresets_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 5).Draw(t, "count")
		type entry struct {
			name, value string
		}
		seen := make(map[string]struct{})
		entries := make([]entry, 0, n)
		for len(entries) < n {
			name := genValidFlagName().Draw(t, "name")
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
			// Values must not contain ';' to avoid splitting ambiguity
			value := rapid.StringMatching(`[^;]*`).Draw(t, "value")
			entries = append(entries, entry{name, value})
		}

		// Build tag string with ';' separator
		parts := make([]string, len(entries))
		for i, e := range entries {
			parts[i] = e.name + "=" + e.value
		}
		tag := strings.Join(parts, ";")

		presets, err := internaltag.ParseFlagPresets(tag)
		if err != nil {
			t.Fatalf("ParseFlagPresets(%q) returned unexpected error: %v", tag, err)
		}
		if len(presets) != len(entries) {
			t.Fatalf("ParseFlagPresets(%q) returned %d presets, expected %d", tag, len(presets), len(entries))
		}
		for i, e := range entries {
			if presets[i].Name != e.name {
				t.Fatalf("preset[%d].Name = %q, expected %q", i, presets[i].Name, e.name)
			}
			// Values are trimmed by ParseFlagPresets
			if presets[i].Value != strings.TrimSpace(e.value) {
				t.Fatalf("preset[%d].Value = %q, expected %q", i, presets[i].Value, strings.TrimSpace(e.value))
			}
		}
		// No duplicate names
		namesSeen := make(map[string]struct{})
		for _, p := range presets {
			if _, dup := namesSeen[p.Name]; dup {
				t.Fatalf("duplicate preset name %q in result", p.Name)
			}
			namesSeen[p.Name] = struct{}{}
		}
	})
}

// P1.3: ParseFlagPresets never panics on arbitrary input.
// If it succeeds, no entry has an empty Name.
func TestProperty_ParseFlagPresets_NoPanic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := genArbitraryPresetTag().Draw(t, "input")
		presets, err := internaltag.ParseFlagPresets(input)
		if err != nil {
			return // errors are fine
		}
		for i, p := range presets {
			if p.Name == "" {
				t.Fatalf("preset[%d] has empty Name for input %q", i, input)
			}
		}
	})
}

// P1.4: When both ';' and ',' are present, ';' wins as separator.
func TestProperty_ParseFlagPresets_SemicolonPrecedence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate 2 valid entries, join with ';', and embed a ',' in a value
		name1 := genValidFlagName().Draw(t, "name1")
		name2 := genValidFlagName().Draw(t, "name2")
		// Ensure distinct names
		if name1 == name2 {
			name2 = name2 + "x"
			if !internaltag.IsValidFlagName(name2) {
				// Skip if we can't make a valid distinct name
				return
			}
		}
		// Value with comma inside — should NOT be treated as separator
		value1 := "a,b"
		value2 := "c"
		tag := name1 + "=" + value1 + ";" + name2 + "=" + value2

		presets, err := internaltag.ParseFlagPresets(tag)
		if err != nil {
			t.Fatalf("ParseFlagPresets(%q) returned error: %v", tag, err)
		}
		if len(presets) != 2 {
			t.Fatalf("expected 2 presets, got %d for input %q", len(presets), tag)
		}
		if presets[0].Name != name1 || presets[0].Value != value1 {
			t.Fatalf("preset[0] = {%q, %q}, expected {%q, %q}", presets[0].Name, presets[0].Value, name1, value1)
		}
		if presets[1].Name != name2 || presets[1].Value != value2 {
			t.Fatalf("preset[1] = {%q, %q}, expected {%q, %q}", presets[1].Name, presets[1].Value, name2, value2)
		}
	})
}

// P1.5: IsMandatory returns true iff the flagrequired tag parses as boolean true.
func TestProperty_IsMandatory_ConsistentWithParseBool(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tagVal := genBoolTagValue().Draw(t, "tagval")

		// Build a struct field with the tag
		typ := reflect.StructOf([]reflect.StructField{
			{
				Name: "F",
				Type: reflect.TypeOf(""),
				Tag:  reflect.StructTag(`flagrequired:"` + tagVal + `"`),
			},
		})
		f := typ.Field(0)

		got := internaltag.IsMandatory(f)
		parsed, parseErr := strconv.ParseBool(tagVal)
		want := parseErr == nil && parsed

		if got != want {
			t.Fatalf("IsMandatory(tag=%q) = %v, expected %v (parseBool=%v, err=%v)", tagVal, got, want, parsed, parseErr)
		}
	})
}
