package flagkit

import (
	"fmt"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
)

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
// register the superset globally and use [OutputFmt.ValidFormat] per command:
//
//	func init() {
//	    // Global: register all formats the CLI knows about
//	    flagkit.RegisterOutputFormats(flagkit.OutputJSON, flagkit.OutputText, flagkit.OutputYAML)
//	}
//
//	// Per-command: validate the subset this command supports
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
