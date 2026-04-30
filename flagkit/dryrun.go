package flagkit

import (
	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
)

func init() {
	registerFlag("dry-run")
}

// DryRun provides a --dry-run flag for safe previewing of operations.
//
// The default is false. When true, commands should describe what they
// would do without making changes. This is agent-friendly: AI agents
// can preview destructive operations before committing.
//
// Usage:
//
//	type Options struct {
//	    flagkit.DryRun
//	}
//
//	// In RunE:
//	if opts.DryRun.Enabled {
//	    fmt.Println("would delete", target)
//	    return nil
//	}
type DryRun struct {
	Enabled bool `flag:"dry-run" flagdescr:"Preview without making changes" default:"false" flagenv:"true"`
}

// Attach implements [structcli.Options].
func (o *DryRun) Attach(c *cobra.Command) error {
	if err := structcli.Define(c, o); err != nil {
		return err
	}

	if f := c.Flags().Lookup("dry-run"); f != nil {
		_ = c.Flags().SetAnnotation("dry-run", FlagKitAnnotation, []string{"true"})
	}

	return nil
}
