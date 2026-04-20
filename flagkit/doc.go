// Package flagkit provides reusable, embeddable flag structs that standardize
// common CLI flag declarations for use with structcli.
//
// Each type encapsulates a single flag with an opinionated name, type, default,
// and description matching industry conventions. This gives CLIs a consistent
// declaration surface — agents and scripts can rely on --follow, --output,
// --timeout, etc. having predictable names and types across tools.
//
// flagkit standardizes flag declarations, not behavioral semantics. How a
// command interprets --quiet or --dry-run is up to the consumer. The value
// is in the shared vocabulary: consistent names, types, and defaults that
// AI agents can recognize across CLIs built with structcli.
//
// # Design Principles
//
//   - One struct per concern — maximum composability
//   - Sensible, agent-friendly defaults (e.g., no auto-tailing, finite timeouts)
//   - Standard flag names matching industry conventions
//   - Works with all structcli features: env vars, config files, JSON Schema,
//     shell completion, and doc generation
//
// # Taxonomy
//
//	Type           Flag          Default  Status
//	─────────────  ────────────  ───────  ────────
//	Follow         --follow/-f   false    available
//	LogLevel       --log-level   info     available
//	ZapLogLevel    --log-level   info     available
//	SlogLogLevel   --log-level   info     available
//	OutputFmt      --output/-o   text     available
//	Verbose        --verbose/-v  0        planned (PR 4)
//	DryRun         --dry-run     false    planned (PR 4)
//	TimeoutOpt     --timeout     30s      planned (PR 5)
//	Quiet          --quiet/-q    false    planned (PR 5)
//
// # Composition
//
// Embed one or more flagkit types in your options struct:
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
package flagkit
