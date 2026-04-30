package flagkit

import (
	"fmt"
	"strings"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
)

// FlagEnumAnnotation mirrors flagEnumAnnotation (unexported, in viper.go)
// from the root structcli package. Duplicated here to avoid an import
// cycle (flagkit → structcli → flagkit).
const FlagEnumAnnotation = "leodido/structcli/flag-enum"

func init() {
	registerFlag("output")
}

// OutputFormat is a string enum for output format selection.
//
// flagkit provides common constants but does NOT auto-register them.
// Call [RegisterOutputFormats] or [structcli.RegisterEnum] in your init()
// to declare the CLI-wide format vocabulary.
type OutputFormat string

const (
	OutputJSON  OutputFormat = "json"
	OutputJSONL OutputFormat = "jsonl"
	OutputText  OutputFormat = "text"
	OutputYAML  OutputFormat = "yaml"
)

// RegisterOutputFormats registers the given output formats for use with
// structcli's enum flag handling. Each format's string value is used as
// both the canonical name and the only accepted alias.
//
// This is a process-global, one-time registration (like [database/sql.Register]).
// Register the superset of all formats your CLI supports. For per-command
// format subsets, use [Output.ValidFormat] in your command's RunE.
//
// Call this in init() before any [structcli.Define] calls:
//
//	func init() {
//	    flagkit.RegisterOutputFormats(flagkit.OutputJSON, flagkit.OutputText, flagkit.OutputYAML)
//	}
//
// For custom aliases, use [structcli.RegisterEnum] directly instead.
func RegisterOutputFormats(formats ...OutputFormat) {
	m := make(map[OutputFormat][]string, len(formats))
	for _, f := range formats {
		m[f] = []string{string(f)}
	}

	structcli.RegisterEnum[OutputFormat](m)
}

// Output provides a --output/-o flag for selecting output format.
//
// The default is text. You must register the supported formats before use
// via [RegisterOutputFormats] or [structcli.RegisterEnum].
//
// For CLIs where different commands support different format subsets,
// register the superset globally, then call [Output.RestrictFormats] after
// Attach. RestrictFormats is the single source of truth: it narrows help,
// JSON Schema, and runtime validation in one call:
//
//	func init() {
//	    flagkit.RegisterOutputFormats(flagkit.OutputJSON, flagkit.OutputText, flagkit.OutputYAML)
//	}
//
//	opts.Attach(cmd)
//	opts.Output.RestrictFormats(cmd, flagkit.OutputJSON, flagkit.OutputYAML)
//
//	// In RunE (no args needed, uses the set from RestrictFormats):
//	if err := opts.Output.ValidFormat(); err != nil {
//	    return err
//	}
type Output struct {
	Format  OutputFormat `flag:"output" flagshort:"o" flagdescr:"Output format" default:"text"`
	allowed []OutputFormat
}

// RestrictFormats narrows the --output flag's help text, enum annotation,
// and runtime validation to only the given formats. Call this after [Attach]
// or [structcli.Define].
//
// This is the single source of truth for per-command format subsets.
// After calling RestrictFormats, [ValidFormat] with no arguments enforces
// the same set, eliminating the need to repeat the allowed list.
//
// Shell completion may still show the globally registered superset because
// cobra does not support overriding completion functions after registration.
//
//	opts.Attach(cmd)
//	opts.Output.RestrictFormats(cmd, flagkit.OutputJSON, flagkit.OutputText)
func (o *Output) RestrictFormats(c *cobra.Command, allowed ...OutputFormat) {
	o.allowed = allowed

	f := c.Flags().Lookup("output")
	if f == nil {
		return
	}

	names := make([]string, len(allowed))
	for i, a := range allowed {
		names[i] = string(a)
	}

	// Update the usage string to show only the allowed subset.
	f.Usage = fmt.Sprintf("Output format {%s}", strings.Join(names, ","))

	// Update the enum annotation used by JSON Schema generation.
	_ = c.Flags().SetAnnotation("output", FlagEnumAnnotation, names)
}

// ValidFormat returns nil if the current output format is allowed, or an
// error describing the mismatch.
//
// When called with no arguments, it validates against the set stored by
// [RestrictFormats]. When called with explicit arguments, it validates
// against those instead (and ignores any stored restriction).
//
// If neither RestrictFormats was called nor explicit arguments are provided,
// ValidFormat returns nil (all formats accepted).
func (o *Output) ValidFormat(allowed ...OutputFormat) error {
	if len(allowed) == 0 {
		allowed = o.allowed
	}
	if len(allowed) == 0 {
		return nil // no restriction
	}

	for _, a := range allowed {
		if o.Format == a {
			return nil
		}
	}

	names := make([]string, len(allowed))
	for i, a := range allowed {
		names[i] = string(a)
	}

	return fmt.Errorf("unsupported output format %q (allowed: %v)", o.Format, names)
}

// Attach implements [structcli.Options].
//
// Returns an error if the --output flag was not created, which typically
// means [RegisterOutputFormats] (or [structcli.RegisterEnum]) was not
// called before Attach.
func (o *Output) Attach(c *cobra.Command) error {
	if err := structcli.Define(c, o); err != nil {
		return err
	}

	f := c.Flags().Lookup("output")
	if f == nil {
		return fmt.Errorf("flagkit: --output flag not created; call RegisterOutputFormats in init() before Attach")
	}

	_ = c.Flags().SetAnnotation("output", FlagKitAnnotation, []string{"true"})

	return nil
}
