package internalhooks

import (
	"fmt"

	"github.com/spf13/cobra"
)

// CompleteHookFunc defines the completion hook for a struct field flag.
type CompleteHookFunc func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

// StoreCompletionHookFuncDirect registers a typed completion hook for a flag.
func StoreCompletionHookFuncDirect(c *cobra.Command, flagName string, complete CompleteHookFunc) {
	if err := c.RegisterFlagCompletionFunc(flagName, cobra.CompletionFunc(complete)); err != nil {
		panic(fmt.Sprintf("structcli: RegisterFlagCompletionFunc(%q) on just-registered flag: %v", flagName, err))
	}
}
