package structcli

import (
	"fmt"
	"reflect"

	internalenv "github.com/leodido/structcli/internal/env"
	internalscope "github.com/leodido/structcli/internal/scope"
	internalvalidation "github.com/leodido/structcli/internal/validation"
	"github.com/spf13/cobra"
)

const (
	bindUsedAnnotation = "leodido/structcli/bind-used"
	// executeCActiveAnnotation uses "executec" (not "execute-c") to match
	// the Go function name ExecuteC as a single token.
	executeCActiveAnnotation = "leodido/structcli/executec-active"
	bindWarnAnnotation = "leodido/structcli/bind-warn-installed"
)

// Bind defines flags from opts on cmd and registers opts for auto-unmarshal
// during [ExecuteC]/[ExecuteOrExit].
//
// opts must be a non-nil struct pointer. If opts implements [Options] (has Attach),
// Attach is called. Otherwise flags are defined directly from struct tags.
//
// Multiple Bind calls per command are supported; unmarshal order matches call order (FIFO).
// Define runs immediately — flags exist on the command after Bind returns.
//
// As a side effect, the first Bind call on a command tree installs a
// [cobra.Command.PersistentPreRunE] on root that warns to stderr when
// the tree is executed via cmd.Execute() instead of [ExecuteC]. This
// warning is best-effort: it is suppressed when a child command defines
// its own PersistentPreRunE (Cobra only runs the nearest ancestor's
// hook), and it can be overwritten if the caller sets root.PersistentPreRunE
// after calling Bind.
func Bind(c *cobra.Command, opts any) error {
	if c == nil {
		return fmt.Errorf("structcli.Bind: command must not be nil")
	}
	if opts == nil {
		return fmt.Errorf("structcli.Bind: opts must not be nil")
	}

	rv := reflect.ValueOf(opts)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("structcli.Bind: opts must be a non-nil struct pointer, got %T", opts)
	}

	// Fast path: if opts implements Options, delegate to Attach.
	// Attach typically calls Define internally, which handles validation,
	// flag definition, viper binding, env binding, and usage setup.
	if o, ok := opts.(Options); ok {
		if err := o.Attach(c); err != nil {
			return fmt.Errorf("structcli.Bind: Attach failed: %w", err)
		}
	} else {
		// Internal define path for plain struct pointers (no Attach method).
		// Replicates the Define sequence: validate → define → BindPFlags → BindEnv → SetupUsage.
		if err := internalvalidation.Struct(c, opts); err != nil {
			return fmt.Errorf("structcli.Bind: %w", err)
		}

		if err := define(c, opts, "", "", nil, false, false, DefaultValidateTagName, DefaultModTagName); err != nil {
			return fmt.Errorf("structcli.Bind: %w", err)
		}

		v := GetViper(c)
		v.BindPFlags(c.Flags())

		if err := internalenv.BindEnv(c); err != nil {
			return fmt.Errorf("structcli.Bind: couldn't bind environment variables: %w", err)
		}

		SetupUsage(c)
	}

	internalscope.Get(c).AddBoundOptions(opts)

	// Mark the root so ExecuteC and the warning hook can detect Bind usage.
	root := c.Root()
	if root.Annotations == nil {
		root.Annotations = make(map[string]string)
	}
	root.Annotations[bindUsedAnnotation] = "true"

	// Install a PersistentPreRunE on root (once per tree) that warns when
	// Bind was used but ExecuteC/ExecuteOrExit was not. The hook is
	// per-tree (no global state) and chains to any existing hook.
	//
	// When ExecuteC is used: it sets executeCActiveAnnotation before
	// calling cmd.ExecuteC(), and prepareTree wraps PersistentPreRunE
	// (saving this hook as the "original"). The pipeline replays it,
	// the hook sees the annotation, and skips the warning.
	//
	// When cmd.Execute() is used directly: prepareTree never runs, so
	// this hook fires as-is. No executeCActiveAnnotation → warning.
	//
	// Limitation: Cobra (without EnableTraverseRunHooks) only runs the
	// nearest ancestor's PersistentPreRunE. If a child command defines
	// its own PersistentPreRunE, root's hook is shadowed and the warning
	// won't fire. This is acceptable — the cmd.Execute() path is already
	// the "wrong" path, and the warning is best-effort.
	if root.Annotations[bindWarnAnnotation] != "true" {
		origPreRunE := root.PersistentPreRunE
		origPreRun := root.PersistentPreRun

		root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
			// Warn only when ExecuteC is not active AND the bind pipeline
			// wrapper was never installed. If a previous ExecuteC call
			// installed the wrapper, auto-unmarshal works even through
			// cmd.Execute() — the wrapper is idempotent and persists.
			if root.Annotations[executeCActiveAnnotation] != "true" &&
				root.Annotations[bindPipelineAnnotation] != "true" {
				root.PrintErrln("Warning: Bind-registered options exist but ExecuteC/ExecuteOrExit was not used.",
					"Bound options will not be auto-unmarshalled. Use structcli.ExecuteC(cmd) or structcli.ExecuteOrExit(cmd) instead of cmd.Execute().")
			}

			if origPreRunE != nil {
				return origPreRunE(cmd, args)
			}
			if origPreRun != nil {
				origPreRun(cmd, args)
			}

			return nil
		}
		root.PersistentPreRun = nil

		root.Annotations[bindWarnAnnotation] = "true"
	}

	return nil
}
