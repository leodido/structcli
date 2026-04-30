// Property-based tests for the Define() path.
//
// Because Define() requires the Options interface (Attach method) and
// reflect.StructOf cannot attach methods, we use concrete option types
// for most properties. Rapid generates field values to exercise edge
// cases in the Define→flag registration pipeline.
//
// P3.13 uses internalvalidation.Struct() on dynamically-built structs to test
// random tag combinations without needing the Options interface.
package define_test

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/leodido/structcli"
	internalenv "github.com/leodido/structcli/internal/env"
	"github.com/leodido/structcli/internal/proptest/gen"
	internalusage "github.com/leodido/structcli/internal/usage"
	internalvalidation "github.com/leodido/structcli/internal/validation"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"pgregory.net/rapid"
)

// --- Concrete option types ---

// allTypesOpts covers every supported primitive, slice, and hook-based type.
type allTypesOpts struct {
	BoolF    bool          `flag:"bool-f"`
	StringF  string        `flag:"string-f"`
	IntF     int           `flag:"int-f"`
	Int8F    int8          `flag:"int8-f"`
	Int16F   int16         `flag:"int16-f"`
	Int32F   int32         `flag:"int32-f"`
	Int64F   int64         `flag:"int64-f"`
	UintF    uint          `flag:"uint-f"`
	Uint8F   uint8         `flag:"uint8-f"`
	Uint16F  uint16        `flag:"uint16-f"`
	Uint32F  uint32        `flag:"uint32-f"`
	Uint64F  uint64        `flag:"uint64-f"`
	Float32F float32       `flag:"float32-f"`
	Float64F float64       `flag:"float64-f"`
	StringsF []string      `flag:"strings-f"`
	IntsF    []int         `flag:"ints-f"`
	DurF     time.Duration `flag:"dur-f"`
	IPF      net.IP        `flag:"ip-f"`
}

func (o *allTypesOpts) Attach(c *cobra.Command) error { return nil }

// allTypesFieldNames returns the flag names for allTypesOpts.
func allTypesFieldNames() []string {
	return []string{
		"bool-f", "string-f", "int-f", "int8-f", "int16-f", "int32-f", "int64-f",
		"uint-f", "uint8-f", "uint16-f", "uint32-f", "uint64-f",
		"float32-f", "float64-f", "strings-f", "ints-f", "dur-f", "ip-f",
	}
}

// hiddenOpts has a field with flaghidden.
type hiddenOpts struct {
	Visible string `flag:"visible"`
	Hidden  string `flag:"hidden" flaghidden:"true"`
}

func (o *hiddenOpts) Attach(c *cobra.Command) error { return nil }

// requiredOpts has a field with flagrequired.
type requiredOpts struct {
	Optional string `flag:"optional"`
	Required string `flag:"required" flagrequired:"true"`
}

func (o *requiredOpts) Attach(c *cobra.Command) error { return nil }

// ignoredOpts has a field with flagignore.
type ignoredOpts struct {
	Active  string `flag:"active"`
	Ignored string `flag:"ignored" flagignore:"true"`
}

func (o *ignoredOpts) Attach(c *cobra.Command) error { return nil }

// groupOpts has fields in different groups.
type groupOpts struct {
	A string `flag:"a" flaggroup:"GroupA"`
	B string `flag:"b" flaggroup:"GroupB"`
	C string `flag:"c"`
}

func (o *groupOpts) Attach(c *cobra.Command) error { return nil }

// defaultOpts has fields with default values.
type defaultOpts struct {
	Name string `flag:"name" default:"world"`
	Port int    `flag:"port" default:"8080"`
}

func (o *defaultOpts) Attach(c *cobra.Command) error { return nil }

// presetOpts has a field with flagpreset.
type presetOpts struct {
	Level int `flag:"level" flagpreset:"verbose=5;quiet=0"`
}

func (o *presetOpts) Attach(c *cobra.Command) error { return nil }

// nestedInner is an inner struct for nesting tests.
type nestedInner struct {
	InnerA string `flag:"inner-a"`
	InnerB int    `flag:"inner-b"`
}

// nestedOpts has a nested struct.
type nestedOpts struct {
	Top  string      `flag:"top"`
	Nest nestedInner `flaggroup:"Nested"`
}

func (o *nestedOpts) Attach(c *cobra.Command) error { return nil }

// envOpts has a field with flagenv.
type envOpts struct {
	Plain   string `flag:"plain"`
	WithEnv string `flag:"with-env" flagenv:"true"`
}

func (o *envOpts) Attach(c *cobra.Command) error { return nil }

// --- Helpers ---

func newCmd() *cobra.Command {
	return &cobra.Command{Use: "test"}
}

func mustDefine(t interface{ Fatal(...any) }, cmd *cobra.Command, opts structcli.Options) {
	if err := structcli.Define(cmd, opts); err != nil {
		t.Fatal("Define failed:", err)
	}
}

func flagCount(cmd *cobra.Command) int {
	n := 0
	cmd.Flags().VisitAll(func(_ *pflag.Flag) { n++ })
	return n
}

// --- Properties ---

// P3.1: Define() never panics on valid option structs with random field values.
// Field-value randomization matters here: NaN, empty strings, negative ints,
// etc. could trigger edge cases in the Define→flag registration pipeline.
func TestProperty_Define_NeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cmd := newCmd()
		opts := &allTypesOpts{
			BoolF:    rapid.Bool().Draw(t, "boolF"),
			StringF:  rapid.String().Draw(t, "stringF"),
			IntF:     rapid.Int().Draw(t, "intF"),
			Int8F:    rapid.Int8().Draw(t, "int8F"),
			Int16F:   rapid.Int16().Draw(t, "int16F"),
			Int32F:   rapid.Int32().Draw(t, "int32F"),
			Int64F:   rapid.Int64().Draw(t, "int64F"),
			UintF:    rapid.Uint().Draw(t, "uintF"),
			Uint8F:   rapid.Uint8().Draw(t, "uint8F"),
			Uint16F:  rapid.Uint16().Draw(t, "uint16F"),
			Uint32F:  rapid.Uint32().Draw(t, "uint32F"),
			Uint64F:  rapid.Uint64().Draw(t, "uint64F"),
			Float32F: rapid.Float32().Draw(t, "float32F"),
			Float64F: rapid.Float64().Draw(t, "float64F"),
		}

		err := structcli.Define(cmd, opts)
		if err != nil {
			t.Fatalf("Define returned error: %v", err)
		}
	})
}

// P3.2: Every non-ignored field has a registered flag.
// Deterministic; no randomization needed.
func TestDefine_AllFieldsHaveFlags(t *testing.T) {
	cmd := newCmd()
	opts := &allTypesOpts{}
	mustDefine(t, cmd, opts)

	for _, name := range allTypesFieldNames() {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("expected flag %q to be registered", name)
		}
	}
}

// P3.3: Flag count matches field count (no presets).
// Deterministic; no randomization needed.
func TestDefine_FlagCountMatchesFieldCount(t *testing.T) {
	cmd := newCmd()
	opts := &allTypesOpts{}
	mustDefine(t, cmd, opts)

	expected := len(allTypesFieldNames())
	got := flagCount(cmd)
	if got != expected {
		t.Fatalf("expected %d flags, got %d", expected, got)
	}
}

// P3.4: flaghidden fields have Hidden == true.
func TestProperty_Define_HiddenFieldsAreHidden(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cmd := newCmd()
		opts := &hiddenOpts{
			Visible: rapid.String().Draw(t, "visible"),
			Hidden:  rapid.String().Draw(t, "hidden"),
		}
		mustDefine(t, cmd, opts)

		visibleFlag := cmd.Flags().Lookup("visible")
		hiddenFlag := cmd.Flags().Lookup("hidden")

		if visibleFlag == nil || hiddenFlag == nil {
			t.Fatal("expected both flags to be registered")
		}
		if visibleFlag.Hidden {
			t.Fatal("visible flag should not be hidden")
		}
		if !hiddenFlag.Hidden {
			t.Fatal("hidden flag should be hidden")
		}
		// Hidden uses pflag.Flag.Hidden bool, not annotations.
		if hiddenFlag.Annotations != nil {
			if _, exists := hiddenFlag.Annotations["hidden"]; exists {
				t.Fatal("unexpected 'hidden' annotation; hidden uses the bool field")
			}
		}
	})
}

// P3.5: flagrequired fields are marked required via annotation.
func TestProperty_Define_RequiredFieldsAreRequired(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cmd := newCmd()
		opts := &requiredOpts{
			Optional: rapid.String().Draw(t, "optional"),
			Required: rapid.String().Draw(t, "required"),
		}
		mustDefine(t, cmd, opts)

		reqFlag := cmd.Flags().Lookup("required")
		if reqFlag == nil {
			t.Fatal("expected 'required' flag to be registered")
		}
		annotations := reqFlag.Annotations
		if annotations == nil {
			t.Fatal("required flag has no annotations")
		}
		if _, ok := annotations[cobra.BashCompOneRequiredFlag]; !ok {
			t.Fatal("required flag missing BashCompOneRequiredFlag annotation")
		}
	})
}

// P3.6: default tag sets the flag's DefValue.
// Deterministic; no randomization needed.
func TestDefine_DefaultTagSetsDefValue(t *testing.T) {
	cmd := newCmd()
	opts := &defaultOpts{}
	mustDefine(t, cmd, opts)

	nameFlag := cmd.Flags().Lookup("name")
	portFlag := cmd.Flags().Lookup("port")

	if nameFlag == nil || portFlag == nil {
		t.Fatal("expected both flags to be registered")
	}
	if nameFlag.DefValue != "world" {
		t.Fatalf("name DefValue = %q, expected %q", nameFlag.DefValue, "world")
	}
	if portFlag.DefValue != "8080" {
		t.Fatalf("port DefValue = %q, expected %q", portFlag.DefValue, "8080")
	}
}

// P3.7: flaggroup annotation is set using the canonical constant.
func TestProperty_Define_GroupAnnotationIsSet(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cmd := newCmd()
		opts := &groupOpts{
			A: rapid.String().Draw(t, "a"),
			B: rapid.String().Draw(t, "b"),
			C: rapid.String().Draw(t, "c"),
		}
		mustDefine(t, cmd, opts)

		checkGroup := func(flagName, expectedGroup string) {
			f := cmd.Flags().Lookup(flagName)
			if f == nil {
				t.Fatalf("expected flag %q to be registered", flagName)
			}
			if expectedGroup == "" {
				return
			}
			ann := f.Annotations
			if ann == nil {
				t.Fatalf("flag %q has no annotations, expected group %q", flagName, expectedGroup)
			}
			groups, ok := ann[internalusage.FlagGroupAnnotation]
			if !ok {
				t.Fatalf("flag %q missing group annotation", flagName)
			}
			if len(groups) != 1 || groups[0] != expectedGroup {
				t.Fatalf("flag %q group = %v, expected [%q]", flagName, groups, expectedGroup)
			}
		}

		checkGroup("a", "GroupA")
		checkGroup("b", "GroupB")
		if cmd.Flags().Lookup("c") == nil {
			t.Fatal("expected flag 'c' to be registered")
		}
	})
}

// P3.8: flagenv fields have env annotations using the canonical constant.
func TestProperty_Define_EnvAnnotationIsSet(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cmd := newCmd()
		opts := &envOpts{
			Plain:   rapid.String().Draw(t, "plain"),
			WithEnv: rapid.String().Draw(t, "withEnv"),
		}
		mustDefine(t, cmd, opts)

		envFlag := cmd.Flags().Lookup("with-env")
		if envFlag == nil {
			t.Fatal("expected 'with-env' flag to be registered")
		}
		ann := envFlag.Annotations
		if ann == nil {
			t.Fatal("with-env flag has no annotations")
		}
		envAnn, ok := ann[internalenv.FlagAnnotation]
		if !ok {
			t.Fatal("with-env flag missing env annotation")
		}
		if len(envAnn) == 0 {
			t.Fatal("with-env flag has empty env annotation")
		}
	})
}

// P3.9: Preset aliases are registered.
func TestProperty_Define_PresetAliasesRegistered(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cmd := newCmd()
		opts := &presetOpts{
			Level: rapid.IntRange(0, 10).Draw(t, "level"),
		}
		mustDefine(t, cmd, opts)

		levelFlag := cmd.Flags().Lookup("level")
		if levelFlag == nil {
			t.Fatal("expected 'level' flag to be registered")
		}

		verboseFlag := cmd.Flags().Lookup("verbose")
		quietFlag := cmd.Flags().Lookup("quiet")
		if verboseFlag == nil {
			t.Fatal("expected 'verbose' preset alias to be registered")
		}
		if quietFlag == nil {
			t.Fatal("expected 'quiet' preset alias to be registered")
		}
	})
}

// P3.10: Preset alias count is additive to flag count.
// Deterministic; no randomization needed.
func TestDefine_PresetAliasCountIsAdditive(t *testing.T) {
	cmd := newCmd()
	opts := &presetOpts{}
	mustDefine(t, cmd, opts)

	got := flagCount(cmd)
	// 1 canonical flag + 2 preset aliases = 3
	if got != 3 {
		t.Fatalf("expected 3 flags (1 + 2 presets), got %d", got)
	}
}

// P3.11: Nested struct fields are flattened and carry the parent's group annotation.
func TestProperty_Define_NestedFieldsFlattened(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cmd := newCmd()
		opts := &nestedOpts{
			Top: rapid.String().Draw(t, "top"),
			Nest: nestedInner{
				InnerA: rapid.String().Draw(t, "innerA"),
				InnerB: rapid.Int().Draw(t, "innerB"),
			},
		}
		mustDefine(t, cmd, opts)

		if cmd.Flags().Lookup("top") == nil {
			t.Fatal("expected 'top' flag")
		}

		innerAFlag := cmd.Flags().Lookup("inner-a")
		innerBFlag := cmd.Flags().Lookup("inner-b")
		if innerAFlag == nil {
			t.Fatal("expected 'inner-a' nested flag")
		}
		if innerBFlag == nil {
			t.Fatal("expected 'inner-b' nested flag")
		}

		// Assert the group annotation IS present, not just correct-when-present.
		for _, f := range []*pflag.Flag{innerAFlag, innerBFlag} {
			if f.Annotations == nil {
				t.Fatalf("flag %q has nil annotations, expected group 'Nested'", f.Name)
			}
			groups, ok := f.Annotations[internalusage.FlagGroupAnnotation]
			if !ok {
				t.Fatalf("flag %q missing group annotation, expected 'Nested'", f.Name)
			}
			if len(groups) != 1 || groups[0] != "Nested" {
				t.Fatalf("flag %q group = %v, expected [Nested]", f.Name, groups)
			}
		}
	})
}

// P3.12: flagignore fields produce no flag.
func TestProperty_Define_IgnoredFieldsProduceNoFlag(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cmd := newCmd()
		opts := &ignoredOpts{
			Active:  rapid.String().Draw(t, "active"),
			Ignored: rapid.String().Draw(t, "ignored"),
		}
		mustDefine(t, cmd, opts)

		if cmd.Flags().Lookup("active") == nil {
			t.Fatal("expected 'active' flag to be registered")
		}
		if cmd.Flags().Lookup("ignored") != nil {
			t.Fatal("expected 'ignored' flag to NOT be registered")
		}
	})
}

// P3.13: Random valid tag combinations pass validation.
//
// Uses internalvalidation.Struct() on dynamically-built structs to test
// the combinatorial tag space. This doesn't require the Options interface.
func TestProperty_Define_RandomTagCombinationsPassValidation(t *testing.T) {
	type tagConfig struct {
		hidden   bool
		required bool
		ignored  bool
		group    string
		defval   string
	}

	rapid.Check(t, func(t *rapid.T) {
		nFields := rapid.IntRange(1, 5).Draw(t, "nFields")
		configs := make([]tagConfig, nFields)
		for i := range configs {
			ignored := rapid.Bool().Draw(t, fmt.Sprintf("ignored_%d", i))
			configs[i] = tagConfig{ignored: ignored}
			if !ignored {
				configs[i].hidden = rapid.Bool().Draw(t, fmt.Sprintf("hidden_%d", i))
				configs[i].required = rapid.Bool().Draw(t, fmt.Sprintf("required_%d", i))
				if rapid.Bool().Draw(t, fmt.Sprintf("hasGroup_%d", i)) {
					configs[i].group = rapid.SampledFrom([]string{"Alpha", "Beta", "Gamma"}).Draw(t, fmt.Sprintf("group_%d", i))
				}
				if rapid.Bool().Draw(t, fmt.Sprintf("hasDefault_%d", i)) {
					configs[i].defval = rapid.StringMatching(`[a-z]{1,5}`).Draw(t, fmt.Sprintf("defval_%d", i))
				}
			}
		}

		// Generate unique flag names.
		flagNames := make([]string, nFields)
		seen := map[string]bool{}
		for i := range flagNames {
			for {
				candidate := gen.ValidFlagName().Draw(t, fmt.Sprintf("flag_%d", i))
				if !seen[candidate] {
					flagNames[i] = candidate
					seen[candidate] = true
					break
				}
			}
		}

		buildTag := func(flagName string, cfg tagConfig) reflect.StructTag {
			parts := []string{fmt.Sprintf(`flag:"%s"`, flagName)}
			if cfg.ignored {
				parts = append(parts, `flagignore:"true"`)
			} else {
				if cfg.hidden {
					parts = append(parts, `flaghidden:"true"`)
				}
				if cfg.required {
					parts = append(parts, `flagrequired:"true"`)
				}
				if cfg.group != "" {
					parts = append(parts, fmt.Sprintf(`flaggroup:"%s"`, cfg.group))
				}
				if cfg.defval != "" {
					parts = append(parts, fmt.Sprintf(`default:"%s"`, cfg.defval))
				}
			}
			return reflect.StructTag(strings.Join(parts, " "))
		}

		// Build a dynamic struct.
		fields := make([]reflect.StructField, nFields)
		for i := range nFields {
			fields[i] = reflect.StructField{
				Name: fmt.Sprintf("F%d", i),
				Type: reflect.TypeOf(""),
				Tag:  buildTag(flagNames[i], configs[i]),
			}
		}
		typ := reflect.StructOf(fields)
		ptr := reflect.New(typ).Interface()

		// Validate the struct through structcli's internal validation.
		cmd := newCmd()
		err := internalvalidation.Struct(cmd, ptr)
		if err != nil {
			t.Fatalf("validation failed for valid tag combination: %v", err)
		}

		// Verify tag metadata is consistent with the generated configs.
		for i, cfg := range configs {
			f := typ.Field(i)
			if cfg.ignored {
				if f.Tag.Get("flagignore") != "true" {
					t.Fatalf("field %d: expected flagignore=true", i)
				}
			}
			if cfg.hidden {
				if f.Tag.Get("flaghidden") != "true" {
					t.Fatalf("field %d: expected flaghidden=true", i)
				}
			}
			if cfg.required {
				if f.Tag.Get("flagrequired") != "true" {
					t.Fatalf("field %d: expected flagrequired=true", i)
				}
			}
			if cfg.group != "" {
				if f.Tag.Get("flaggroup") != cfg.group {
					t.Fatalf("field %d: expected flaggroup=%q, got %q", i, cfg.group, f.Tag.Get("flaggroup"))
				}
			}
			if cfg.defval != "" {
				if f.Tag.Get("default") != cfg.defval {
					t.Fatalf("field %d: expected default=%q, got %q", i, cfg.defval, f.Tag.Get("default"))
				}
			}
		}
	})
}
