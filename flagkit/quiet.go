package flagkit

import (
	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
)

func init() {
	registerFlag("quiet")
}

// Quiet provides a --quiet/-q flag for suppressing non-essential output.
//
// The default is false. When true, commands should only emit machine-readable
// output (e.g., IDs, status codes) and suppress progress messages, banners,
// and decorative formatting.
//
// Usage:
//
//	type Options struct {
//	    flagkit.Quiet
//	}
//
//	// In RunE:
//	if !opts.Quiet.Enabled {
//	    fmt.Println("Deploying to production...")
//	}
type Quiet struct {
	Enabled bool `flag:"quiet" flagshort:"q" flagdescr:"Suppress non-essential output" default:"false"`
}

// Attach implements [structcli.Options].
func (o *Quiet) Attach(c *cobra.Command) error {
	if err := structcli.Define(c, o); err != nil {
		return err
	}

	if f := c.Flags().Lookup("quiet"); f != nil {
		_ = c.Flags().SetAnnotation("quiet", FlagKitAnnotation, []string{"true"})
	}

	return nil
}
