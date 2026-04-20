package flagkit

import (
	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
)

func init() {
	registerFlag("verbose")
}

// Verbose provides a --verbose/-v count flag for verbosity levels.
//
// The default is 0 (quiet). Each -v increments the count: -v is 1, -vv is 2,
// -vvv is 3. This is the standard Unix convention for verbosity.
//
// Usage:
//
//	type Options struct {
//	    flagkit.Verbose
//	}
//
//	// In RunE:
//	if opts.Verbose.Level > 1 {
//	    // extra debug output
//	}
type Verbose struct {
	Level int `flag:"verbose" flagshort:"v" flagtype:"count" flagdescr:"Increase verbosity (-v, -vv, -vvv)" default:"0"`
}

// Attach implements [structcli.Options].
func (o *Verbose) Attach(c *cobra.Command) error {
	if err := structcli.Define(c, o); err != nil {
		return err
	}

	if f := c.Flags().Lookup("verbose"); f != nil {
		_ = c.Flags().SetAnnotation("verbose", FlagKitAnnotation, []string{"true"})
	}

	return nil
}
