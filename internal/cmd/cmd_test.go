package internalcmd

import (
	"testing"

	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecursivelyWrapRun_SkipsRunWhenDebugActive(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	runCalled := false
	run := &cobra.Command{
		Use: "run",
		Run: func(_ *cobra.Command, _ []string) {
			runCalled = true
		},
	}
	root.AddCommand(run)
	internalscope.Get(root).Viper().Set("debug-options", true)

	RecursivelyWrapRun(root)
	root.SetArgs([]string{"run"})
	err := root.Execute()
	require.NoError(t, err)
	assert.False(t, runCalled)
}

func TestRecursivelyWrapRun_SkipsRunEWhenDebugActive(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	runECalled := false
	run := &cobra.Command{
		Use: "run",
		RunE: func(_ *cobra.Command, _ []string) error {
			runECalled = true
			return nil
		},
	}
	root.AddCommand(run)
	internalscope.Get(root).Viper().Set("debug-options", true)

	RecursivelyWrapRun(root)
	root.SetArgs([]string{"run"})
	err := root.Execute()
	require.NoError(t, err)
	assert.False(t, runECalled)
}

func TestRecursivelyWrapRun_ExecutesWhenDebugInactive(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	runCalled := false
	runECalled := false
	run := &cobra.Command{
		Use: "run",
		Run: func(_ *cobra.Command, _ []string) {
			runCalled = true
		},
	}
	runE := &cobra.Command{
		Use: "rune",
		RunE: func(_ *cobra.Command, _ []string) error {
			runECalled = true
			return nil
		},
	}
	root.AddCommand(run, runE)

	RecursivelyWrapRun(root)

	root.SetArgs([]string{"run"})
	require.NoError(t, root.Execute())
	assert.True(t, runCalled)

	root.SetArgs([]string{"rune"})
	require.NoError(t, root.Execute())
	assert.True(t, runECalled)
}

func TestRecursivelyWrapRun_IsIdempotentAndRecursive(t *testing.T) {
	root := &cobra.Command{Use: "app"}
	mid := &cobra.Command{Use: "mid"}
	leafCalls := 0
	leaf := &cobra.Command{
		Use: "leaf",
		Run: func(_ *cobra.Command, _ []string) {
			leafCalls++
		},
	}
	root.AddCommand(mid)
	mid.AddCommand(leaf)

	RecursivelyWrapRun(root)
	RecursivelyWrapRun(root)

	assert.Equal(t, "true", root.Annotations[wrappedRunAnnotation])
	assert.Equal(t, "true", mid.Annotations[wrappedRunAnnotation])
	assert.Equal(t, "true", leaf.Annotations[wrappedRunAnnotation])

	root.SetArgs([]string{"mid", "leaf"})
	require.NoError(t, root.Execute())
	assert.Equal(t, 1, leafCalls)
}
