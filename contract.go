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
// Does not require Options (Attach) — works with plain struct pointers via Bind.
type Validatable interface {
	Validate(context.Context) []error
}

// Transformable is a struct that supports transformation after unmarshalling.
//
// Transform is called automatically during Unmarshal(), before Validate.
// Does not require Options (Attach) — works with plain struct pointers via Bind.
type Transformable interface {
	Transform(context.Context) error
}

// ContextInjector is a struct that propagates values into the command context after unmarshalling.
//
// Context is called automatically during Unmarshal() to derive a new context.
// Does not require Options (Attach) — works with plain struct pointers via Bind.
//
// FromContext (reading values back from context) is a user-side pattern, not part of this interface.
type ContextInjector interface {
	Context(context.Context) context.Context
}

// ValidatableOptions extends Options with validation capabilities.
//
// Deprecated: Use Validatable instead. ValidatableOptions requires implementing Attach,
// which is unnecessary for validation. Validatable works with both Options implementors
// and plain struct pointers.
type ValidatableOptions interface {
	Options
	Validate(context.Context) []error
}

// TransformableOptions extends Options with transformation capabilities.
//
// Deprecated: Use Transformable instead. TransformableOptions requires implementing Attach,
// which is unnecessary for transformation. Transformable works with both Options implementors
// and plain struct pointers.
type TransformableOptions interface {
	Options
	Transform(context.Context) error
}

// EnumValuer is an optional interface that pflag.Value implementations can satisfy
// to declare their allowed values at the type level.
//
// When a pflag.Value returned by a DefineHookFunc (built-in or custom) implements
// EnumValuer, structcli stores the allowed values as a flag annotation during Define().
// This is the authoritative source of enum values — no description string parsing needed.
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
// Deprecated: Use ContextInjector instead. ContextInjector only requires the Context method
// (propagation). FromContext is a user-side pattern — structcli never calls it internally.
// ContextInjector works with both Options implementors and plain struct pointers.
type ContextOptions interface {
	Options
	Context(context.Context) context.Context
	FromContext(context.Context) error
}
