package internalcmd

import (
	"sync"

	internaldebug "github.com/leodido/structcli/internal/debug"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ExecutionInterceptor struct {
	Annotation      string
	ShouldIntercept func(*cobra.Command) bool
	Intercept       func(*cobra.Command, []string) (bool, error)
}

const (
	wrappedRunAnnotation           = "structcli/debug-run-wrapped"
	interceptedExecutionAnnotation = "structcli/execution-intercepted"
)

var (
	// Intercepted executions are tracked process-wide because cobra.OnFinalize
	// also runs process-wide. Registering RestoreInterceptedExecutions once is
	// enough: the shared finalizer drains this whole map after each Execute call,
	// so independent command trees in the same process still get their original
	// flag state restored before the next execution begins.
	interceptedExecutionsMu sync.Mutex
	interceptedExecutions   = map[*cobra.Command]bool{}
	finalizeOnce            sync.Once
)

func PrepareInterceptedExecution(c *cobra.Command) {
	if c == nil {
		return
	}

	finalizeOnce.Do(func() {
		cobra.OnFinalize(RestoreInterceptedExecutions)
	})

	interceptedExecutionsMu.Lock()
	if _, ok := interceptedExecutions[c]; !ok {
		interceptedExecutions[c] = c.DisableFlagParsing
	}
	interceptedExecutionsMu.Unlock()

	c.DisableFlagParsing = true
	if c.Annotations == nil {
		c.Annotations = make(map[string]string)
	}
	c.Annotations[interceptedExecutionAnnotation] = "true"
}

func IsExecutionIntercepted(c *cobra.Command) bool {
	return c != nil && c.Annotations != nil && c.Annotations[interceptedExecutionAnnotation] == "true"
}

func RestoreInterceptedExecutions() {
	interceptedExecutionsMu.Lock()
	defer interceptedExecutionsMu.Unlock()

	for c, originalDisableFlagParsing := range interceptedExecutions {
		if c == nil {
			continue
		}

		resetFlags(c.Root())
		c.DisableFlagParsing = originalDisableFlagParsing
		if c.Annotations != nil {
			delete(c.Annotations, interceptedExecutionAnnotation)
		}
	}

	clear(interceptedExecutions)
}

func resetFlags(root *cobra.Command) {
	if root == nil {
		return
	}

	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, fs := range []*pflag.FlagSet{c.LocalFlags(), c.PersistentFlags()} {
			if fs == nil {
				continue
			}
			fs.VisitAll(func(f *pflag.Flag) {
				_ = f.Value.Set(f.DefValue)
				f.Changed = false
			})
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}

func RecursivelyWrapExecution(c *cobra.Command, interceptor ExecutionInterceptor) {
	if c == nil {
		return
	}
	if c.Annotations == nil {
		c.Annotations = make(map[string]string)
	}
	if interceptor.Annotation != "" && c.Annotations[interceptor.Annotation] == "true" {
		return
	}

	wrapArgs(c, interceptor.ShouldIntercept)
	wrapPersistentPreRun(c)
	wrapPreRun(c, interceptor.Intercept)
	wrapRun(c)
	wrapPostRun(c)
	wrapPersistentPostRun(c)

	if interceptor.Annotation != "" {
		c.Annotations[interceptor.Annotation] = "true"
	}

	for _, sub := range c.Commands() {
		RecursivelyWrapExecution(sub, interceptor)
	}
}

func wrapArgs(c *cobra.Command, shouldIntercept func(*cobra.Command) bool) {
	originalArgs := c.Args
	c.Args = func(cmd *cobra.Command, args []string) error {
		if shouldIntercept != nil && shouldIntercept(cmd) {
			PrepareInterceptedExecution(cmd)
			return nil
		}
		if originalArgs != nil {
			return originalArgs(cmd, args)
		}
		return cobra.ArbitraryArgs(cmd, args)
	}
}

func wrapPersistentPreRun(c *cobra.Command) {
	if c.PersistentPreRunE == nil && c.PersistentPreRun == nil {
		return
	}

	originalPersistentPreRunE := c.PersistentPreRunE
	originalPersistentPreRun := c.PersistentPreRun
	c.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if IsExecutionIntercepted(cmd) {
			return nil
		}
		if originalPersistentPreRunE != nil {
			return originalPersistentPreRunE(cmd, args)
		}
		if originalPersistentPreRun != nil {
			originalPersistentPreRun(cmd, args)
		}
		return nil
	}
	c.PersistentPreRun = nil
}

func wrapPreRun(c *cobra.Command, intercept func(*cobra.Command, []string) (bool, error)) {
	originalPreRunE := c.PreRunE
	originalPreRun := c.PreRun
	c.PreRunE = func(cmd *cobra.Command, args []string) error {
		if intercept != nil {
			handled, err := intercept(cmd, args)
			if err != nil {
				return err
			}
			if handled {
				PrepareInterceptedExecution(cmd)
				return nil
			}
		}

		if originalPreRunE != nil {
			return originalPreRunE(cmd, args)
		}
		if originalPreRun != nil {
			originalPreRun(cmd, args)
		}
		return nil
	}
	c.PreRun = nil
}

func wrapRun(c *cobra.Command) {
	if c.RunE == nil && c.Run == nil {
		return
	}

	originalRunE := c.RunE
	originalRun := c.Run
	c.RunE = func(cmd *cobra.Command, args []string) error {
		if IsExecutionIntercepted(cmd) {
			return nil
		}
		if originalRunE != nil {
			return originalRunE(cmd, args)
		}
		if originalRun != nil {
			originalRun(cmd, args)
		}
		return nil
	}
	c.Run = nil
}

func wrapPostRun(c *cobra.Command) {
	if c.PostRunE == nil && c.PostRun == nil {
		return
	}

	originalPostRunE := c.PostRunE
	originalPostRun := c.PostRun
	c.PostRunE = func(cmd *cobra.Command, args []string) error {
		if IsExecutionIntercepted(cmd) {
			return nil
		}
		if originalPostRunE != nil {
			return originalPostRunE(cmd, args)
		}
		if originalPostRun != nil {
			originalPostRun(cmd, args)
		}
		return nil
	}
	c.PostRun = nil
}

func wrapPersistentPostRun(c *cobra.Command) {
	if c.PersistentPostRunE == nil && c.PersistentPostRun == nil {
		return
	}

	originalPersistentPostRunE := c.PersistentPostRunE
	originalPersistentPostRun := c.PersistentPostRun
	c.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		if IsExecutionIntercepted(cmd) {
			return nil
		}
		if originalPersistentPostRunE != nil {
			return originalPersistentPostRunE(cmd, args)
		}
		if originalPersistentPostRun != nil {
			originalPersistentPostRun(cmd, args)
		}
		return nil
	}
	c.PersistentPostRun = nil
}

func RecursivelyWrapRun(c *cobra.Command) {
	if c.Annotations == nil || c.Annotations[wrappedRunAnnotation] != "true" {
		// Idempotency guard: SetupDebug can trigger wrapping multiple times via cobra.OnInitialize.
		if c.RunE != nil {
			originalRunE := c.RunE
			c.RunE = func(c *cobra.Command, args []string) error {
				if internaldebug.IsDebugActive(c) {
					return nil // Exit cleanly without running the original function
				}
				return originalRunE(c, args)
			}
		} else if c.Run != nil {
			// Handle non-error returning Run as well
			originalRun := c.Run
			c.Run = func(c *cobra.Command, args []string) {
				if internaldebug.IsDebugActive(c) {
					return // Exit cleanly
				}
				originalRun(c, args)
			}
		}

		if c.Annotations == nil {
			c.Annotations = make(map[string]string)
		}
		c.Annotations[wrappedRunAnnotation] = "true"
	}

	// Recurse into subcommands
	for _, sub := range c.Commands() {
		RecursivelyWrapRun(sub)
	}
}
