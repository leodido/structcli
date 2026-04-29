package structcli

import (
	"fmt"
	"strings"
	"sync"

	internalcmd "github.com/leodido/structcli/internal/cmd"
	internalconfig "github.com/leodido/structcli/internal/config"
	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
)

const bindPipelineAnnotation = "leodido/structcli/bind-pipeline-wrapped"

// hookSet holds the original PersistentPreRunE/PersistentPreRun hooks
// saved before wrapping, keyed by command pointer.
type hookSet struct {
	preRunE map[*cobra.Command]func(*cobra.Command, []string) error
	preRun  map[*cobra.Command]func(*cobra.Command, []string)
}

// hookStore is a process-safe map from root command → hookSet.
// Created once per command tree on the first ExecuteC call and reused
// across repeated executions. Entries are removed by Reset().
var hookStore sync.Map // *cobra.Command → *hookSet

// configOnceStore holds the per-execution sync.Once for config auto-load.
// Replaced on every ExecuteC call so repeated executions reload config.
var configOnceStore sync.Map // *cobra.Command → *sync.Once

func getHooks(root *cobra.Command) *hookSet {
	val, _ := hookStore.Load(root)
	if val == nil {
		return nil
	}

	return val.(*hookSet)
}

func getConfigOnce(root *cobra.Command) *sync.Once {
	val, _ := configOnceStore.Load(root)
	if val == nil {
		return nil
	}

	return val.(*sync.Once)
}

// ExecuteC prepares the command tree for execution and delegates to cmd.ExecuteC().
//
// Preparation (idempotent — safe to call multiple times on the same tree):
//   - Sets SilenceErrors and SilenceUsage on the root command.
//   - Runs SetupUsage on every command in the tree.
//   - Recursively wraps PersistentPreRunE on every command to run the bind pipeline
//     (auto-unmarshal for all Bind-registered options, root-to-leaf, FIFO per command).
//   - When WithConfig was used in Setup, auto-loads config (UseConfigSimple) once
//     before the first auto-unmarshal.
//   - Skips the bind pipeline when execution is intercepted (--jsonschema, --mcp).
//   - Preserves any user-set PersistentPreRunE or PersistentPreRun.
//   - Warns (once per tree) if non-leaf commands have Bind-registered local flags
//     but root.TraverseChildren is false.
//   - Suppresses the [Bind] warning hook by setting an annotation that is cleared
//     after execution returns.
//
// Returns the executed subcommand and any error.
func ExecuteC(cmd *cobra.Command) (*cobra.Command, error) {
	root := cmd.Root()

	root.SilenceErrors = true
	root.SilenceUsage = true

	// Hook storage is created once per command tree and reused across
	// repeated ExecuteC calls. The hookStore entry is keyed by root
	// command pointer; it is populated during the first prepareTree
	// and persists for the tree's lifetime.
	hookStore.LoadOrStore(root, &hookSet{
		preRunE: make(map[*cobra.Command]func(*cobra.Command, []string) error),
		preRun:  make(map[*cobra.Command]func(*cobra.Command, []string)),
	})

	// Fresh once-guard per ExecuteC call so config is reloaded on each
	// execution. The wrapper closures look this up at runtime via
	// getConfigOnce rather than capturing a stale pointer.
	configOnceStore.Store(root, &sync.Once{})

	prepareTree(root)

	warnTraverseChildren(root)

	// Signal that ExecuteC is active so the Bind warning hook
	// (installed by Bind as a PersistentPreRunE) knows not to fire.
	if root.Annotations == nil {
		root.Annotations = make(map[string]string)
	}
	root.Annotations[executeCActiveAnnotation] = "true"
	// Clear after execution so a subsequent cmd.Execute() on the same tree
	// is not silently treated as an ExecuteC call.
	defer delete(root.Annotations, executeCActiveAnnotation)

	return cmd.ExecuteC()
}

const traverseChildrenWarnAnnotation = "leodido/structcli/traverse-children-warned"

// warnTraverseChildren prints a diagnostic (once per tree) when non-leaf
// commands have Bind-registered local flags but the root's TraverseChildren
// is false. Without TraverseChildren, Cobra does not parse ancestor local
// flags when a subcommand is invoked, so bound local flags on parent
// commands would be rejected as unknown.
func warnTraverseChildren(root *cobra.Command) {
	if root.TraverseChildren {
		return
	}
	if root.Annotations != nil && root.Annotations[traverseChildrenWarnAnnotation] == "true" {
		return
	}

	cmds := commandsWithBoundOptionsAndChildren(root)
	if len(cmds) == 0 {
		return
	}

	paths := make([]string, len(cmds))
	for i, c := range cmds {
		paths[i] = fmt.Sprintf("%q", c.CommandPath())
	}

	if len(cmds) == 1 {
		root.PrintErrln(fmt.Sprintf("Warning: command %s has Bind-registered local flags and subcommands, but TraverseChildren is false.", paths[0]),
			"Set TraverseChildren = true on the root command, or bind shared options on each leaf command.")
	} else {
		root.PrintErrln(fmt.Sprintf("Warning: commands %s have Bind-registered local flags and subcommands, but TraverseChildren is false.", strings.Join(paths, ", ")),
			"Set TraverseChildren = true on the root command, or bind shared options on each leaf command.")
	}

	if root.Annotations == nil {
		root.Annotations = make(map[string]string)
	}
	root.Annotations[traverseChildrenWarnAnnotation] = "true"
}

// commandsWithBoundOptionsAndChildren returns commands that have both
// bound options and at least one subcommand.
func commandsWithBoundOptionsAndChildren(c *cobra.Command) []*cobra.Command {
	var result []*cobra.Command
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		subs := cmd.Commands()
		if len(subs) > 0 && len(internalscope.Get(cmd).BoundOptions()) > 0 {
			result = append(result, cmd)
		}
		for _, sub := range subs {
			walk(sub)
		}
	}
	walk(c)

	return result
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
	hooks := getHooks(c.Root())
	if hooks != nil {
		if c.PersistentPreRunE != nil {
			hooks.preRunE[c] = c.PersistentPreRunE
		}
		if c.PersistentPreRun != nil {
			hooks.preRun[c] = c.PersistentPreRun
		}
	}

	c.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Skip pipeline on intercepted execution (--jsonschema, --mcp, etc.)
		if internalcmd.IsExecutionIntercepted(cmd) {
			return nil
		}

		// Auto-load config once per execution if WithConfig was used.
		// Look up the current configOnce at runtime so repeated ExecuteC
		// calls get a fresh guard (not the one captured at wrap time).
		if once := getConfigOnce(cmd.Root()); once != nil {
			if err := autoLoadConfig(cmd, once); err != nil {
				return err
			}
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

// autoLoadConfig calls UseConfigSimple once per execution when WithConfig
// was used in Setup. The sync.Once ensures config is loaded exactly once
// even though every command in the tree has the wrapper.
//
// After loading, it merges config into each ancestor command's viper so
// that ancestor-bound options see config values during unmarshal.
func autoLoadConfig(cmd *cobra.Command, configOnce *sync.Once) error {
	root := cmd.Root()
	if root.Annotations == nil || root.Annotations[configAutoLoadAnnotation] != "true" {
		return nil
	}

	var configErr error
	configOnce.Do(func() {
		_, message, err := UseConfigSimple(cmd)
		if err != nil {
			configErr = err

			return
		}
		if message != "" {
			cmd.Println(message)
		}

		// UseConfigSimple merges config into the executed command's viper.
		// Also merge into each ancestor's viper so ancestor-bound options
		// see config values when unmarshalled with the owner command.
		rootVip := internalscope.Get(root).ConfigViper()
		for c := cmd.Parent(); c != nil; c = c.Parent() {
			configToMerge := internalconfig.Merge(rootVip.AllSettings(), c)
			if mergeErr := internalscope.Get(c).Viper().MergeConfigMap(configToMerge); mergeErr != nil {
				configErr = fmt.Errorf("error merging config for command %q: %w", c.CommandPath(), mergeErr)

				return
			}
		}
	})
	if configErr != nil {
		return fmt.Errorf("structcli: auto-load config: %w", configErr)
	}

	return nil
}

// replayOriginalHooks walks from root to the executed command and calls
// any original PersistentPreRunE or PersistentPreRun that was saved
// before wrapping, in root-first order.
func replayOriginalHooks(executedCmd *cobra.Command, args []string) error {
	hooks := getHooks(executedCmd.Root())
	if hooks == nil {
		return nil
	}

	path := pathToRoot(executedCmd)

	for _, c := range path {
		if hook, ok := hooks.preRunE[c]; ok {
			if err := hook(executedCmd, args); err != nil {
				return err
			}
		} else if hook, ok := hooks.preRun[c]; ok {
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

	// Track unmarshalled opts pointers to avoid re-unmarshalling the same
	// struct when it is bound to multiple commands in the ancestor chain.
	// The first (ancestor-most) binding wins because the pipeline walks
	// root-first and TraverseChildren parses ancestor flags first.
	seen := make(map[any]bool)

	for _, c := range path {
		scope := internalscope.Get(c)
		for _, opts := range scope.BoundOptions() {
			if seen[opts] {
				continue
			}
			seen[opts] = true
			// Unmarshal using the owner command (c) for viper/flag resolution,
			// but inject context on the executed command so descendants see it.
			if err := unmarshalForPipeline(c, executedCmd, opts); err != nil {
				return err
			}
		}
	}

	return nil
}

// unmarshalForPipeline unmarshals opts using ownerCmd for viper/flag resolution,
// then re-injects context on executedCmd so descendants can see ContextInjector
// values. Cobra does not propagate SetContext calls made on ancestors after
// command resolution, so context must be set on the executed command.
func unmarshalForPipeline(ownerCmd, executedCmd *cobra.Command, opts any) error {
	if err := unmarshal(ownerCmd, opts); err != nil {
		return err
	}

	// If context was injected on ownerCmd but executedCmd is different,
	// re-inject on executedCmd so descendants see the value.
	if ownerCmd != executedCmd {
		if o, ok := opts.(ContextInjector); ok {
			executedCmd.SetContext(o.Context(executedCmd.Context()))
		} else if o, ok := opts.(ContextOptions); ok {
			executedCmd.SetContext(o.Context(executedCmd.Context()))
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
