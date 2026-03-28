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

// ValidatableOptions extends Options with validation capabilities.
//
// The Validate method is called automatically during Unmarshal().
type ValidatableOptions interface {
	Options
	Validate(context.Context) []error
}

// TransformableOptions extends Options with transformation capabilities.
//
// The Transform method is called automatically during Unmarshal() before validation.
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
// The Context method is called automatically during Unmarshal() to modify the command context.
type ContextOptions interface {
	Options
	Context(context.Context) context.Context
	FromContext(context.Context) error
}
