package structcli

import (
	"fmt"
	"reflect"

	internalhooks "github.com/leodido/structcli/internal/hooks"
)

// TypeHooks defines custom flag behavior for a type.
//
// Define creates the pflag.Value for this type. Called during Define/Bind for
// each struct field of type T. Receives the specific field's value and metadata.
//
// Decode converts raw input (string from env/config) to T during Unmarshal.
type TypeHooks[T any] struct {
	Define DefineHookFunc
	Decode DecodeHookFunc
}

// RegisterType registers custom flag hooks for type T.
//
// After registration, struct fields of type T work without any special tag
// or interface. The define hook is called once per field during Define/Bind;
// the decode hook is called during Unmarshal for env/config values.
//
// Must be called in init() before any Define/Bind calls.
// Panics if T is already registered (duplicate or conflict with a built-in).
// Panics if Define is nil. Panics if Decode is nil.
func RegisterType[T any](hooks TypeHooks[T]) {
	typeName := reflect.TypeFor[T]().String()

	if hooks.Define == nil {
		panic(fmt.Sprintf("structcli: RegisterType[%s]: Define hook must not be nil", typeName))
	}
	if hooks.Decode == nil {
		panic(fmt.Sprintf("structcli: RegisterType[%s]: Decode hook must not be nil", typeName))
	}

	if _, exists := internalhooks.DefineHookRegistry[typeName]; exists {
		panic(fmt.Sprintf("structcli: RegisterType[%s]: type is already registered", typeName))
	}

	internalhooks.DefineHookRegistry[typeName] = hooks.Define

	internalhooks.RegisterUserDecodeHook(typeName, hooks.Decode)
}
