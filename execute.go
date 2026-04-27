package structcli

import (
	internalcmd "github.com/leodido/structcli/internal/cmd"
	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
)

const bindPipelineAnnotation = "structcli/bind-pipeline-wrapped"

// originalHookKey is used to store original persistent hooks before wrapping.
const originalPreRunEKey = "structcli/original-persistent-prerun-e"
const originalPreRunKey = "structcli/original-persistent-prerun"

// originalHooks stores the original PersistentPreRunE/PersistentPreRun per command,
// keyed by command pointer. We use a separate map because annotations are string-only.
var originalHooks = struct {
	preRunE map[*cobra.Command]func(*cobra.Command, []string) error
	preRun  map[*cobra.Command]func(*cobra.Command, []string)
}{
	preRunE: make(map[*cobra.Command]func(*cobra.Command, []string) error),
	preRun:  make(map[*cobra.Command]func(*cobra.Command, []string)),
}

// ExecuteC prepares the command tree for execution and delegates to cmd.ExecuteC().
//
// Preparation (idempotent — safe to call multiple times on the same tree):
//   - Sets SilenceErrors and SilenceUsage on the root command.
//   - Runs SetupUsage on every command in the tree.
//   - Recursively wraps PersistentPreRunE on every command to run the bind pipeline
//     (auto-unmarshal for all Bind-registered options, root-to-leaf, FIFO per command).
//   - Skips the bind pipeline when execution is intercepted (--jsonschema, --mcp).
//   - Preserves any user-set PersistentPreRunE or PersistentPreRun.
//
// Returns the executed subcommand and any error.
func ExecuteC(cmd *cobra.Command) (*cobra.Command, error) {
	root := cmd.Root()

	root.SilenceErrors = true
	root.SilenceUsage = true

	prepareTree(root)

	return cmd.ExecuteC()
}

// prepareTree walks the command tree and installs the bind pipeline wrapper
// and SetupUsage on every command. Idempotent via annotation guard.
func prepareTree(c *cobra.Command) {
	if c == nil {
		return
	}

	SetupUsage(c)
	wrapBindPipeline(c)

	for _, sub := range c.Commands() {
		prepareTree(sub)
	}
}

// wrapBindPipeline installs a PersistentPreRunE on c that runs the bind
// pipeline before chaining to original hooks. Idempotent.
//
// Because Cobra (without EnableTraverseRunHooks) only executes the first
// PersistentPreRunE it finds walking from the executed command upward,
// every command in the tree gets a wrapper. The wrapper itself runs the
// full root-to-leaf pipeline and then replays all original ancestor
// persistent hooks in root-first order. This ensures user hooks on
// ancestor commands fire even when a descendant's wrapper is the one
// Cobra picks.
func wrapBindPipeline(c *cobra.Command) {
	if c.Annotations == nil {
		c.Annotations = make(map[string]string)
	}
	if c.Annotations[bindPipelineAnnotation] == "true" {
		return
	}

	// Save original hooks before overwriting.
	if c.PersistentPreRunE != nil {
		originalHooks.preRunE[c] = c.PersistentPreRunE
	}
	if c.PersistentPreRun != nil {
		originalHooks.preRun[c] = c.PersistentPreRun
	}

	c.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Skip pipeline on intercepted execution (--jsonschema, --mcp, etc.)
		if internalcmd.IsExecutionIntercepted(cmd) {
			return nil
		}

		// Run bind pipeline: walk root → executed command, unmarshal bound options.
		if err := runBindPipeline(cmd); err != nil {
			return err
		}

		// Replay original persistent hooks from root to the command whose
		// wrapper Cobra selected (which is cmd's closest ancestor with a
		// PersistentPreRunE — i.e., this command c, since we wrapped it).
		// We replay all ancestors' original hooks in root-first order so
		// user hooks on parent commands still fire.
		if err := replayOriginalHooks(cmd, args); err != nil {
			return err
		}

		return nil
	}
	c.PersistentPreRun = nil

	c.Annotations[bindPipelineAnnotation] = "true"
}

// replayOriginalHooks walks from root to the executed command and calls
// any original PersistentPreRunE or PersistentPreRun that was saved
// before wrapping, in root-first order.
func replayOriginalHooks(executedCmd *cobra.Command, args []string) error {
	path := pathToRoot(executedCmd)

	for _, c := range path {
		if hook, ok := originalHooks.preRunE[c]; ok {
			if err := hook(executedCmd, args); err != nil {
				return err
			}
		} else if hook, ok := originalHooks.preRun[c]; ok {
			hook(executedCmd, args)
		}
	}

	return nil
}

// runBindPipeline walks from root to the executed command, collecting bound
// options from each command's scope, and calls unmarshal for each in FIFO order.
//
// Every Unmarshal call receives the executed command (not the owning command),
// because Unmarshal rebuilds flag metadata by walking from the passed command
// upward through its ancestors.
func runBindPipeline(executedCmd *cobra.Command) error {
	path := pathToRoot(executedCmd)

	for _, c := range path {
		scope := internalscope.Get(c)
		for _, opts := range scope.BoundOptions() {
			if err := unmarshal(executedCmd, opts); err != nil {
				return err
			}
		}
	}

	return nil
}

// pathToRoot returns the path from root to cmd (root-first order).
func pathToRoot(cmd *cobra.Command) []*cobra.Command {
	var path []*cobra.Command
	for c := cmd; c != nil; c = c.Parent() {
		path = append(path, c)
	}
	// Reverse to get root-first order.
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path
}
