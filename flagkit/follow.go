package flagkit

import (
	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
)

// FlagKitAnnotation is the pflag annotation key set on flags defined by
// flagkit types. The generate package uses this to detect flagkit usage
// and emit development guidance in generated docs.
const FlagKitAnnotation = "___leodido_structcli_flagkit"

// flagKitFlags is the registry of flag names owned by flagkit types.
// Each type registers its flag name via registerFlag in its own file's init().
var flagKitFlags []string

// registerFlag adds a flag name to the flagkit registry.
// Called by each type's init() function.
func registerFlag(name string) {
	flagKitFlags = append(flagKitFlags, name)
}

// AnnotateCommand marks all flagkit-owned flags on the command with the
// [FlagKitAnnotation]. Call this after [structcli.Define] when embedding
// flagkit types in a parent struct.
//
// When using a flagkit type standalone via its Attach method, the
// annotation is set automatically and this call is not needed.
// For embedded usage, [structcli.Define] traverses into the embedded
// struct but does not call its Attach method, so AnnotateCommand
// must be called explicitly to set the annotation.
func AnnotateCommand(c *cobra.Command) {
	for _, name := range flagKitFlags {
		if f := c.Flags().Lookup(name); f != nil {
			_ = c.Flags().SetAnnotation(name, FlagKitAnnotation, []string{"true"})
		}
	}
}

func init() {
	registerFlag("follow")
}

// Follow provides a --follow/-f boolean flag for opt-in streaming.
//
// When false (the default), commands should print current output and exit.
// When true, commands should stream output continuously. This default is
// agent-friendly — AI agents and scripts won't hang on indefinite tailing.
//
// Usage:
//
//	type LogOptions struct {
//	    flagkit.Follow
//	    Service string `flag:"service" flagdescr:"Service name"`
//	}
//
//	func (o *LogOptions) Attach(c *cobra.Command) error {
//	    if err := structcli.Define(c, o); err != nil {
//	        return err
//	    }
//	    flagkit.AnnotateCommand(c)
//	    return nil
//	}
//
//	// In RunE:
//	if opts.Follow.Enabled {
//	    streamLogs(ctx)
//	} else {
//	    printCurrentLogs()
//	}
type Follow struct {
	Enabled bool `flag:"follow" flagshort:"f" flagdescr:"Stream output continuously" default:"false"`
}

// Attach implements [structcli.Options].
func (o *Follow) Attach(c *cobra.Command) error {
	if err := structcli.Define(c, o); err != nil {
		return err
	}

	if f := c.Flags().Lookup("follow"); f != nil {
		_ = c.Flags().SetAnnotation("follow", FlagKitAnnotation, []string{"true"})
	}

	return nil
}
