package internalhooks

import "github.com/spf13/pflag"

// enumFlagWrapper wraps a pflag.Value and attaches the known enum values.
// This satisfies the structcli.EnumValuer interface, enabling define.go to
// store allowed values as flag annotations without parsing description strings.
type enumFlagWrapper struct {
	pflag.Value
	values []string
}

// EnumValues returns the allowed string values for this enum flag.
func (w *enumFlagWrapper) EnumValues() []string {
	return w.values
}

// WrapWithEnumValues wraps a pflag.Value to also satisfy structcli.EnumValuer.
func WrapWithEnumValues(v pflag.Value, values []string) pflag.Value {
	return &enumFlagWrapper{Value: v, values: values}
}
