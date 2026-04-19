package values

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/pflag"
)

// EnumStringValue implements pflag.Value for string-based enum types.
// It validates on Set() that the value is in the allowed set, supports
// case-insensitive matching, and exposes allowed values via EnumValues().
type EnumStringValue[E ~string] struct {
	target    *E
	allowed   map[string]E // lowercase input → enum constant
	canonical []string     // sorted canonical names (first name per constant)
}

// NewEnumString creates an EnumStringValue for the given target pointer.
// values maps each enum constant to its string representations; the first
// string in each slice is canonical, additional strings are aliases.
func NewEnumString[E ~string](target *E, values map[E][]string) *EnumStringValue[E] {
	allowed := make(map[string]E, len(values)*2)
	canonical := make([]string, 0, len(values))
	for enumVal, names := range values {
		if len(names) == 0 {
			continue
		}
		canonical = append(canonical, names[0])
		for _, name := range names {
			allowed[strings.ToLower(name)] = enumVal
		}
	}
	slices.Sort(canonical)

	return &EnumStringValue[E]{
		target:    target,
		allowed:   allowed,
		canonical: canonical,
	}
}

func (v *EnumStringValue[E]) String() string {
	if v.target == nil {
		return ""
	}

	return string(*v.target)
}

func (v *EnumStringValue[E]) Set(s string) error {
	val, ok := v.allowed[strings.ToLower(s)]
	if !ok {
		return fmt.Errorf("invalid value %q (allowed: %s)", s, strings.Join(v.canonical, ", "))
	}
	*v.target = val

	return nil
}

func (v *EnumStringValue[E]) Type() string {
	return "string"
}

// EnumValues returns the sorted canonical names for this enum.
// Satisfies the structcli.EnumValuer interface.
func (v *EnumStringValue[E]) EnumValues() []string {
	return v.canonical
}

var _ pflag.Value = (*EnumStringValue[string])(nil)
