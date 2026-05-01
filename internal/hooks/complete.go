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
// Panics if the flag does not exist (structurally impossible when called after
// flag registration) or if completeM is invalid.
func StoreCompletionHookFunc(c *cobra.Command, flagName string, completeM reflect.Value) {
	if !completeM.IsValid() {
		panic(fmt.Sprintf("structcli: invalid completion hook for flag %q", flagName))
	}

	if err := c.RegisterFlagCompletionFunc(flagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		results := completeM.Call([]reflect.Value{
			reflect.ValueOf(cmd),
			reflect.ValueOf(args),
			reflect.ValueOf(toComplete),
		})

		suggestions, _ := results[0].Interface().([]string)
		directive, _ := results[1].Interface().(cobra.ShellCompDirective)

		return suggestions, directive
	}); err != nil {
		panic(fmt.Sprintf("structcli: RegisterFlagCompletionFunc(%q) on just-registered flag: %v", flagName, err))
	}
}

// StoreCompletionHookFuncDirect registers a typed completion hook for a flag.
// Unlike StoreCompletionHookFunc, it calls the function directly without
// reflect.Value.Call, enabling dead-code elimination.
func StoreCompletionHookFuncDirect(c *cobra.Command, flagName string, fn CompleteHookFunc) {
	if err := c.RegisterFlagCompletionFunc(flagName, cobra.CompletionFunc(fn)); err != nil {
		panic(fmt.Sprintf("structcli: RegisterFlagCompletionFunc(%q) on just-registered flag: %v", flagName, err))
	}
}
