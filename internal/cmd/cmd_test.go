package internalcmd

import (
	"bytes"
	"errors"
	"fmt"
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
	internalscope.Get(root).Viper().Set("debug-options", "text")

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
	internalscope.Get(root).Viper().Set("debug-options", "text")

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

func TestPrepareInterceptedExecution_RestoresFlagsAndState(t *testing.T) {
	RestoreInterceptedExecutions()
	t.Cleanup(RestoreInterceptedExecutions)

	root := &cobra.Command{Use: "app"}
	root.PersistentFlags().String("config", "default.yaml", "config file")

	srv := &cobra.Command{Use: "srv"}
	srv.Flags().Int("port", 8080, "server port")
	root.AddCommand(srv)

	require.NoError(t, root.PersistentFlags().Set("config", "custom.yaml"))
	require.NoError(t, srv.Flags().Set("port", "9090"))

	assert.True(t, root.PersistentFlags().Lookup("config").Changed)
	assert.True(t, srv.Flags().Lookup("port").Changed)

	PrepareInterceptedExecution(srv)

	assert.True(t, IsExecutionIntercepted(srv))
	assert.True(t, srv.DisableFlagParsing)

	RestoreInterceptedExecutions()

	config, err := root.PersistentFlags().GetString("config")
	require.NoError(t, err)
	port, err := srv.Flags().GetInt("port")
	require.NoError(t, err)

	assert.Equal(t, "default.yaml", config)
	assert.Equal(t, 8080, port)
	assert.False(t, root.PersistentFlags().Lookup("config").Changed)
	assert.False(t, srv.Flags().Lookup("port").Changed)
	assert.False(t, srv.DisableFlagParsing)
	assert.False(t, IsExecutionIntercepted(srv))
}

func TestRecursivelyWrapExecution_PreservesLifecycleWhenNotIntercepted(t *testing.T) {
	var calls []string

	root := &cobra.Command{
		Use: "app",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			calls = append(calls, "persistent-pre")
		},
		PersistentPostRun: func(_ *cobra.Command, _ []string) {
			calls = append(calls, "persistent-post")
		},
	}

	srv := &cobra.Command{
		Use: "srv",
		Args: func(_ *cobra.Command, args []string) error {
			calls = append(calls, "args")
			if len(args) != 1 {
				return fmt.Errorf("expected one argument, got %d", len(args))
			}
			return nil
		},
		PreRun: func(_ *cobra.Command, _ []string) {
			calls = append(calls, "pre")
		},
		Run: func(_ *cobra.Command, _ []string) {
			calls = append(calls, "run")
		},
		PostRun: func(_ *cobra.Command, _ []string) {
			calls = append(calls, "post")
		},
	}
	root.AddCommand(srv)

	RecursivelyWrapExecution(root, ExecutionInterceptor{
		Annotation: "leodido/structcli/test-wrapped",
		ShouldIntercept: func(_ *cobra.Command) bool {
			return false
		},
		Intercept: func(_ *cobra.Command, _ []string) (bool, error) {
			calls = append(calls, "intercept")
			return false, nil
		},
	})

	root.SetArgs([]string{"srv", "now"})
	require.NoError(t, root.Execute())

	assert.Equal(t, []string{
		"args",
		"persistent-pre",
		"intercept",
		"pre",
		"run",
		"post",
		"persistent-post",
	}, calls)
	assert.Equal(t, "true", root.Annotations["leodido/structcli/test-wrapped"])
	assert.Equal(t, "true", srv.Annotations["leodido/structcli/test-wrapped"])
}

func TestRecursivelyWrapExecution_InterceptsWithoutRunningCommand(t *testing.T) {
	t.Run("handled", func(t *testing.T) {
		RestoreInterceptedExecutions()
		t.Cleanup(RestoreInterceptedExecutions)

		var calls []string
		root := &cobra.Command{
			Use: "app",
			PersistentPreRun: func(_ *cobra.Command, _ []string) {
				calls = append(calls, "persistent-pre")
			},
			PreRun: func(_ *cobra.Command, _ []string) {
				calls = append(calls, "pre")
			},
			Run: func(_ *cobra.Command, _ []string) {
				calls = append(calls, "run")
			},
			PostRun: func(_ *cobra.Command, _ []string) {
				calls = append(calls, "post")
			},
			PersistentPostRun: func(_ *cobra.Command, _ []string) {
				calls = append(calls, "persistent-post")
			},
		}

		RecursivelyWrapExecution(root, ExecutionInterceptor{
			Annotation: "leodido/structcli/test-wrapped",
			Intercept: func(_ *cobra.Command, _ []string) (bool, error) {
				calls = append(calls, "intercept")
				return true, nil
			},
		})

		require.NoError(t, root.Execute())
		assert.Equal(t, []string{"persistent-pre", "intercept"}, calls)
		assert.False(t, IsExecutionIntercepted(root))
	})

	t.Run("error", func(t *testing.T) {
		root := &cobra.Command{
			Use: "app",
			Run: func(_ *cobra.Command, _ []string) {
				t.Fatal("run should not execute when interception fails")
			},
		}

		RecursivelyWrapExecution(root, ExecutionInterceptor{
			Annotation: "leodido/structcli/test-wrapped",
			Intercept: func(_ *cobra.Command, _ []string) (bool, error) {
				return false, errors.New("boom")
			},
		})

		err := root.Execute()
		require.Error(t, err)
		assert.EqualError(t, err, "boom")
	})
}

func TestEnsureRunnable_InterceptsCommandWithoutRunE(t *testing.T) {
	RestoreInterceptedExecutions()
	t.Cleanup(RestoreInterceptedExecutions)

	intercepted := false
	// Root has no RunE/Run — cobra would normally short-circuit to Help().
	root := &cobra.Command{
		Use:           "app",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	EnsureRunnable(root)
	RecursivelyWrapExecution(root, ExecutionInterceptor{
		Annotation:      "leodido/structcli/test-wrapped",
		ShouldIntercept: func(_ *cobra.Command) bool { return true },
		Intercept: func(_ *cobra.Command, _ []string) (bool, error) {
			intercepted = true
			return true, nil
		},
	})

	require.NoError(t, root.Execute())
	assert.True(t, intercepted, "interception should fire on commands without RunE")
}

func TestEnsureRunnable_ShowsHelpWhenNotIntercepted(t *testing.T) {
	RestoreInterceptedExecutions()
	t.Cleanup(RestoreInterceptedExecutions)

	// Command without RunE/Run should show help when not intercepted.
	root := &cobra.Command{
		Use:   "app",
		Short: "test app",
	}

	EnsureRunnable(root)
	RecursivelyWrapExecution(root, ExecutionInterceptor{
		Annotation:      "leodido/structcli/test-wrapped",
		ShouldIntercept: func(_ *cobra.Command) bool { return false },
		Intercept: func(_ *cobra.Command, _ []string) (bool, error) {
			return false, nil
		},
	})

	var out bytes.Buffer
	root.SetOut(&out)
	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "test app", "should show help output")
}

func TestEnsureRunnable_NoopWhenRunEExists(t *testing.T) {
	called := false
	root := &cobra.Command{
		Use: "app",
		RunE: func(cmd *cobra.Command, args []string) error {
			called = true
			return nil
		},
	}

	EnsureRunnable(root)
	require.NoError(t, root.Execute())
	assert.True(t, called, "original RunE should still be called")
}
