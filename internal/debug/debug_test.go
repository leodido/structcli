package internaldebug

import (
	"testing"

	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDebugActive_DefaultFlag(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	run := &cobra.Command{Use: "run"}
	root.AddCommand(run)
	root.PersistentFlags().Bool("debug-options", false, "")

	require.NoError(t, root.PersistentFlags().Set("debug-options", "true"))
	assert.True(t, IsDebugActive(run))
}

func TestIsDebugActive_CustomFlagAnnotation(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	run := &cobra.Command{
		Use:         "run",
		Annotations: map[string]string{FlagAnnotation: "debug-mode"},
	}
	root.AddCommand(run)
	root.PersistentFlags().Bool("debug-mode", false, "")

	require.NoError(t, root.PersistentFlags().Set("debug-mode", "true"))
	assert.True(t, IsDebugActive(run))
}

func TestIsDebugActive_FromScopedViper(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	run := &cobra.Command{Use: "run"}
	root.AddCommand(run)

	internalscope.Get(root).Viper().Set("debug-options", true)
	assert.True(t, IsDebugActive(run))
}

func TestIsDebugActive_FalseWhenUnset(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	run := &cobra.Command{Use: "run"}
	root.AddCommand(run)

	assert.False(t, IsDebugActive(run))
}
