package flagkit

import (
	"time"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
)

func init() {
	registerFlag("timeout")
}

// Timeout provides a --timeout flag for operation deadlines.
//
// The default is 30s. Accepts any value parseable by [time.ParseDuration].
// This is agent-friendly — operations won't hang indefinitely.
//
// Usage:
//
//	type Options struct {
//	    flagkit.TimeoutOpt
//	}
//
//	// In RunE:
//	ctx, cancel := context.WithTimeout(ctx, opts.TimeoutOpt.Duration)
//	defer cancel()
type TimeoutOpt struct {
	Duration time.Duration `flag:"timeout" flagdescr:"Operation timeout" default:"30s" flagenv:"true"`
}

// Attach implements [structcli.Options].
func (o *TimeoutOpt) Attach(c *cobra.Command) error {
	if err := structcli.Define(c, o); err != nil {
		return err
	}

	if f := c.Flags().Lookup("timeout"); f != nil {
		_ = c.Flags().SetAnnotation("timeout", FlagKitAnnotation, []string{"true"})
	}

	return nil
}
