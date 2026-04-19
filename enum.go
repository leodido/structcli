package structcli

import (
	"fmt"
	"reflect"

	internalhooks "github.com/leodido/structcli/internal/hooks"
)

// RegisterEnum registers a string-based enum type for automatic flag handling.
// After registration, struct fields of type E work without flagcustom:"true"
// or manual Define/Decode/Complete hook methods.
//
// values maps each enum constant to its string representations. The first
// string in each slice is the canonical name shown in help text and shell
// completion; additional strings are accepted as aliases during parsing
// (case-insensitive). Canonical names appear sorted alphabetically in help text.
//
// Must be called in init() before any Define() calls. Panics if the type
// is already registered (duplicate registration or conflict with a built-in),
// or if values is empty.
//
// Example:
//
//	type Environment string
//	const (
//	    EnvDev  Environment = "dev"
//	    EnvProd Environment = "prod"
//	)
//
//	func init() {
//	    structcli.RegisterEnum[Environment](map[Environment][]string{
//	        EnvDev:  {"dev", "development"},
//	        EnvProd: {"prod", "production"},
//	    })
//	}
// RegisterIntEnum registers an integer-based enum type for automatic flag handling.
// Same semantics as RegisterEnum but for types with an integer underlying type
// (e.g., custom iota-based enums). Uses enumflag/v2 internally for flag parsing.
//
// Must be called in init() before any Define() calls. Panics if the type
// is already registered.
//
// Example:
//
//	type Priority int
//	const (
//	    PriorityLow    Priority = 0
//	    PriorityMedium Priority = 1
//	    PriorityHigh   Priority = 2
//	)
//
//	func init() {
//	    structcli.RegisterIntEnum[Priority](map[Priority][]string{
//	        PriorityLow:    {"low"},
//	        PriorityMedium: {"medium", "med"},
//	        PriorityHigh:   {"high", "hi"},
//	    })
//	}
func RegisterIntEnum[E ~int | ~int8 | ~int16 | ~int32 | ~int64](values map[E][]string) {
	typeName := reflect.TypeFor[E]().String()

	if _, exists := internalhooks.DefineHookRegistry[typeName]; exists {
		panic(fmt.Sprintf("structcli: RegisterIntEnum: type %q is already registered", typeName))
	}

	internalhooks.DefineHookRegistry[typeName] = internalhooks.DefineIntEnumHookFunc(values)

	annName := fmt.Sprintf("StringTo%sHookFunc", typeName)
	internalhooks.RegisterDecodeHook(typeName, annName, internalhooks.StringToIntEnumHookFunc(values))
}

func RegisterEnum[E ~string](values map[E][]string) {
	if len(values) == 0 {
		panic("structcli: RegisterEnum: values must not be empty")
	}

	typeName := reflect.TypeFor[E]().String()

	if _, exists := internalhooks.DefineHookRegistry[typeName]; exists {
		panic(fmt.Sprintf("structcli: RegisterEnum: type %q is already registered", typeName))
	}

	// Register define hook
	internalhooks.DefineHookRegistry[typeName] = internalhooks.DefineStringEnumHookFunc(values)

	// Register decode hook (panics on duplicate annotation name)
	annName := fmt.Sprintf("StringTo%sHookFunc", typeName)
	internalhooks.RegisterDecodeHook(typeName, annName, internalhooks.StringToEnumHookFunc(values))
}
