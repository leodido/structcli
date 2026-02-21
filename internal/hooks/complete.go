package internalhooks

import (
	"fmt"
	"reflect"

	"github.com/spf13/cobra"
)

// CompleteHookFunc defines the optional completion hook for a struct field flag.
//
// Methods matching the signature and naming convention `Complete<FieldName>`
// are discovered during Define() and automatically registered on the command.
type CompleteHookFunc func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

// StoreCompletionHookFunc registers a validated completion hook method for a flag.
func StoreCompletionHookFunc(c *cobra.Command, flagName string, completeM reflect.Value) error {
	if !completeM.IsValid() {
		return fmt.Errorf("invalid completion hook for flag %q", flagName)
	}

	return c.RegisterFlagCompletionFunc(flagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		results := completeM.Call([]reflect.Value{
			reflect.ValueOf(cmd),
			reflect.ValueOf(args),
			reflect.ValueOf(toComplete),
		})

		suggestions, _ := results[0].Interface().([]string)
		directive, _ := results[1].Interface().(cobra.ShellCompDirective)

		return suggestions, directive
	})
}
