package structcli

import (
	"context"

	"github.com/spf13/cobra"
)

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
