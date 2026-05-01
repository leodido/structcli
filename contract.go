package structcli

import (
	"context"

	internalhooks "github.com/leodido/structcli/internal/hooks"
	"github.com/spf13/cobra"
)

// DefineHookFunc defines how to create a pflag.Value for a custom type
// during Define/Bind.
type DefineHookFunc = internalhooks.DefineHookFunc

// DecodeHookFunc defines how to decode a raw value into a custom type
// during Unmarshal.
type DecodeHookFunc = internalhooks.DecodeHookFunc

// CompleteHookFunc defines how to provide shell completion candidates
// for a flag.
type CompleteHookFunc = internalhooks.CompleteHookFunc

// FieldHook bundles the Define and Decode hooks for a single struct field.
//
// Both hooks are optional: if Define is nil the field falls through to the
// type registry or built-in handling; if Decode is nil the default decode
// path is used.
type FieldHook struct {
	// Define creates the pflag.Value for this field.
	Define DefineHookFunc

	// Decode converts raw input to the field's type during Unmarshal.
	Decode DecodeHookFunc
}

// FieldHookProvider provides per-field Define/Decode hooks.
//
// Implement this interface when the same type needs different flag behavior
// in different fields, or when a standard type needs custom handling for a
// specific field.
//
// Map keys are struct field names (e.g., "ListenAddr", not the flag name
// "listen"). Unknown keys that do not match any struct field cause an error
// at Define/Bind time.
//
// Precedence: FieldHookProvider > [RegisterType] > built-in registry.
type FieldHookProvider interface {
	FieldHooks() map[string]FieldHook
}

// FieldCompleter provides per-field shell completion hooks.
//
// Map keys are struct field names. Works for any field that becomes a flag,
// not only fields with [FieldHookProvider] hooks.
//
// If a completion function is already registered on a flag before Define,
// structcli preserves it (the FieldCompleter hook is not applied).
type FieldCompleter interface {
	CompletionHooks() map[string]CompleteHookFunc
}

// Options represents a struct that can define command-line flags, env vars, config file keys.
//
// Types implementing this interface can be used with Define() to automatically generate flags from struct fields.
type Options interface {
	Attach(*cobra.Command) error
}

// Validatable is a struct that supports validation after unmarshalling.
//
// Validate is called automatically during Unmarshal(), after Transform.
// Does not require [Options] (Attach). Works with plain struct pointers via Bind.
type Validatable interface {
	Validate(context.Context) []error
}

// Transformable is a struct that supports transformation after unmarshalling.
//
// Transform is called automatically during Unmarshal(), before Validate.
// Does not require [Options] (Attach). Works with plain struct pointers via Bind.
type Transformable interface {
	Transform(context.Context) error
}

// ContextInjector is a struct that propagates values into the command context after unmarshalling.
//
// Context is called automatically during Unmarshal() to derive a new context.
// Does not require [Options] (Attach). Works with plain struct pointers via Bind.
//
// Reading values back from context (FromContext) is a user-side pattern,
// not part of this interface.
type ContextInjector interface {
	Context(context.Context) context.Context
}

// ValidatableOptions extends Options with validation capabilities.
//
// For the Bind API, consider using [Validatable] instead. It does not
// require implementing Attach and works with plain struct pointers.
// ValidatableOptions remains the right choice when using Define/Unmarshal
// directly or when the type already implements [Options].
type ValidatableOptions interface {
	Options
	Validate(context.Context) []error
}

// TransformableOptions extends Options with transformation capabilities.
//
// For the Bind API, consider using [Transformable] instead. It does not
// require implementing Attach and works with plain struct pointers.
// TransformableOptions remains the right choice when using Define/Unmarshal
// directly or when the type already implements [Options].
type TransformableOptions interface {
	Options
	Transform(context.Context) error
}

// EnumValuer is an optional interface that pflag.Value implementations can
// satisfy to declare their allowed values at the type level.
//
// When a pflag.Value returned by a DefineHookFunc (built-in or custom)
// implements EnumValuer, structcli stores the allowed values as a flag
// annotation during Define(). This is the authoritative source of enum
// values; no description string parsing is needed.
//
// Example:
//
//	type myEnumFlag struct {
//	    pflag.Value          // embed the underlying pflag.Value
//	    allowed []string
//	}
//	func (f *myEnumFlag) EnumValues() []string { return f.allowed }
type EnumValuer interface {
	EnumValues() []string
}

// ContextOptions extends Options with context manipulation capabilities.
//
// For the Bind API, consider using [ContextInjector] instead. It only
// requires the Context method (propagation) and works with plain struct
// pointers. FromContext is a user-side pattern; structcli never calls it
// internally. ContextOptions remains the right choice when using
// Define/Unmarshal directly or when the type already implements [Options].
type ContextOptions interface {
	Options
	Context(context.Context) context.Context
	FromContext(context.Context) error
}
