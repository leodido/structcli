package flagkit

import (
	"fmt"
	"strings"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
)

// flagEnumAnnotation mirrors the structcli annotation key for enum values.
const flagEnumAnnotation = "___leodido_structcli_flagenum"

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
// register the superset globally, then call [OutputFmt.RestrictFormats] after
// Attach to narrow help/completion/schema, and [OutputFmt.ValidFormat] in RunE:
//
//	func init() {
//	    // Global: register all formats the CLI knows about
//	    flagkit.RegisterOutputFormats(flagkit.OutputJSON, flagkit.OutputText, flagkit.OutputYAML)
//	}
//
//	// Per-command: restrict help/completion/schema, then validate at runtime
//	opts.Attach(cmd)
//	opts.OutputFmt.RestrictFormats(cmd, flagkit.OutputJSON, flagkit.OutputYAML)
//
//	func (o *ExportOptions) RunE(cmd *cobra.Command, args []string) error {
//	    if err := o.ValidFormat(flagkit.OutputJSON, flagkit.OutputYAML); err != nil {
//	        return err
//	    }
//	    // ...
//	}
type OutputFmt struct {
	Format OutputFormat `flag:"output" flagshort:"o" flagdescr:"Output format" default:"text"`
}

// ValidFormat returns nil if the current output format is one of the allowed
// formats, or an error describing the mismatch. Use this for per-command
// format validation when different commands support different subsets.
func (o *OutputFmt) ValidFormat(allowed ...OutputFormat) error {
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

// RestrictFormats narrows the --output flag's help text and enum annotation
// to only the given formats. Call this after [Attach] or [structcli.Define]
// to make the command's declared contract (help, JSON Schema) match what
// [ValidFormat] will accept at runtime.
//
// Shell completion may still show the globally registered superset because
// cobra does not support overriding completion functions after registration.
//
//	opts.Attach(cmd)
//	opts.RestrictFormats(cmd, flagkit.OutputJSON, flagkit.OutputText)
func (o *OutputFmt) RestrictFormats(c *cobra.Command, allowed ...OutputFormat) {
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
	_ = c.Flags().SetAnnotation("output", flagEnumAnnotation, names)
}

// Attach implements [structcli.Options].
func (o *OutputFmt) Attach(c *cobra.Command) error {
	if err := structcli.Define(c, o); err != nil {
		return err
	}

	if f := c.Flags().Lookup("output"); f != nil {
		_ = c.Flags().SetAnnotation("output", FlagKitAnnotation, []string{"true"})
	}

	return nil
}
