package structcli

import (
	"fmt"
	"reflect"

	internalenv "github.com/leodido/structcli/internal/env"
	internalscope "github.com/leodido/structcli/internal/scope"
	internalvalidation "github.com/leodido/structcli/internal/validation"
	"github.com/spf13/cobra"
)

// Bind defines flags from opts on cmd and registers opts for auto-unmarshal
// during ExecuteC/ExecuteOrExit.
//
// opts must be a non-nil struct pointer. If opts implements Options (has Attach),
// Attach is called. Otherwise flags are defined directly from struct tags.
//
// Multiple Bind calls per command are supported; unmarshal order matches call order (FIFO).
// Define runs immediately — flags exist on the command after Bind returns.
//
// The current manual Unmarshal model still works. Auto-unmarshal via the execution
// pipeline is wired in ExecuteC (PR 3).
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

	return nil
}
