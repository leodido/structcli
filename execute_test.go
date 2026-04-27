package structcli

import (
	"context"
	"fmt"
	"testing"

	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test option types ---

type execPlainOpts struct {
	Port int    `flag:"port" flagshort:"p" flagdescr:"server port" default:"3000"`
	Host string `flag:"host" flagdescr:"server host" default:"localhost"`
}

type execAttachOpts struct {
	Verbose bool `flag:"verbose" flagshort:"v" flagdescr:"verbose output"`
}

func (o *execAttachOpts) Attach(c *cobra.Command) error {
	return Define(c, o)
}

type execContextOpts struct {
	AppName string `flag:"app-name" flagdescr:"application name" default:"myapp"`
}

func (o *execContextOpts) Context(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey("app-name"), o.AppName)
}

type ctxKey string

type execChildOpts struct {
	Debug bool `flag:"debug" flagshort:"d" flagdescr:"enable debug"`
}

// --- Tests ---

func TestExecuteC_BasicExecution(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	var ran bool
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			ran = true
			return nil
		},
	}

	c, err := ExecuteC(cmd)
	require.NoError(t, err)
	assert.True(t, ran)
	assert.Equal(t, "test", c.Name())
}

func TestExecuteC_AutoUnmarshal_PlainStruct(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	opts := &execPlainOpts{}
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}

	require.NoError(t, Bind(cmd, opts))

	cmd.SetArgs([]string{"--port", "8080", "--host", "0.0.0.0"})
	_, err := ExecuteC(cmd)
	require.NoError(t, err)

	assert.Equal(t, 8080, opts.Port)
	assert.Equal(t, "0.0.0.0", opts.Host)
}

func TestExecuteC_AutoUnmarshal_OptionsImplementor(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	opts := &execAttachOpts{}
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}

	require.NoError(t, Bind(cmd, opts))

	cmd.SetArgs([]string{"--verbose"})
	_, err := ExecuteC(cmd)
	require.NoError(t, err)

	assert.True(t, opts.Verbose)
}

func TestExecuteC_AutoUnmarshal_Defaults(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	opts := &execPlainOpts{}
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}

	require.NoError(t, Bind(cmd, opts))

	cmd.SetArgs([]string{})
	_, err := ExecuteC(cmd)
	require.NoError(t, err)

	assert.Equal(t, 3000, opts.Port)
	assert.Equal(t, "localhost", opts.Host)
}

func TestExecuteC_PopulatedBeforePreRunE(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	opts := &execPlainOpts{}
	var portInPreRun int

	cmd := &cobra.Command{
		Use: "test",
		PreRunE: func(c *cobra.Command, args []string) error {
			portInPreRun = opts.Port
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}

	require.NoError(t, Bind(cmd, opts))

	cmd.SetArgs([]string{"--port", "9090"})
	_, err := ExecuteC(cmd)
	require.NoError(t, err)

	assert.Equal(t, 9090, portInPreRun, "opts should be populated before PreRunE")
}

func TestExecuteC_MultipleBind_FIFOOrder(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	plain := &execPlainOpts{}
	attach := &execAttachOpts{}

	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}

	require.NoError(t, Bind(cmd, plain))
	require.NoError(t, Bind(cmd, attach))

	cmd.SetArgs([]string{"--port", "4000", "--verbose"})
	_, err := ExecuteC(cmd)
	require.NoError(t, err)

	assert.Equal(t, 4000, plain.Port)
	assert.True(t, attach.Verbose)
}

func TestExecuteC_AncestorBeforeDescendant(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	rootOpts := &execContextOpts{}
	childOpts := &execChildOpts{}

	var ctxValueInRun string

	root := &cobra.Command{Use: "root"}
	child := &cobra.Command{
		Use: "child",
		RunE: func(c *cobra.Command, args []string) error {
			// rootOpts.Context() should have been called before childOpts unmarshal
			if v := c.Context().Value(ctxKey("app-name")); v != nil {
				ctxValueInRun = v.(string)
			}
			return nil
		},
	}
	root.AddCommand(child)

	// Bind rootOpts on child too — flags are local per command.
	// The ancestor-before-descendant contract is about unmarshal order
	// when both root and child have bound options.
	require.NoError(t, Bind(root, rootOpts))
	require.NoError(t, Bind(child, childOpts))

	// rootOpts flags are on root (local), childOpts flags are on child (local).
	// Execute child with child-local flags only; root opts get defaults.
	root.SetArgs([]string{"child", "--debug"})
	_, err := ExecuteC(root)
	require.NoError(t, err)

	assert.Equal(t, "myapp", rootOpts.AppName, "root opts should get default value")
	assert.True(t, childOpts.Debug)
	assert.Equal(t, "myapp", ctxValueInRun, "root context injection should be visible in child RunE")
}

func TestExecuteC_PreservesUserPersistentPreRunE(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	opts := &execPlainOpts{}
	var userHookRan bool
	var portInUserHook int

	cmd := &cobra.Command{
		Use: "root",
	}
	cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		userHookRan = true
		portInUserHook = opts.Port
		return nil
	}

	child := &cobra.Command{
		Use: "child",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.AddCommand(child)

	// Bind on child so flags are local to child
	require.NoError(t, Bind(child, opts))

	cmd.SetArgs([]string{"child", "--port", "5555"})
	_, err := ExecuteC(cmd)
	require.NoError(t, err)

	assert.True(t, userHookRan, "user PersistentPreRunE should still run")
	assert.Equal(t, 5555, portInUserHook, "opts should be populated before user hook")
}

func TestExecuteC_PreservesUserPersistentPreRun(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	opts := &execPlainOpts{}
	var userHookRan bool

	cmd := &cobra.Command{
		Use: "root",
	}
	cmd.PersistentPreRun = func(c *cobra.Command, args []string) {
		userHookRan = true
	}

	child := &cobra.Command{
		Use: "child",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.AddCommand(child)

	// Bind on child so flags are local to child
	require.NoError(t, Bind(child, opts))

	cmd.SetArgs([]string{"child", "--port", "5555"})
	_, err := ExecuteC(cmd)
	require.NoError(t, err)

	assert.True(t, userHookRan, "user PersistentPreRun should still run")
}

func TestExecuteC_SilencesErrorsAndUsage(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}

	_, err := ExecuteC(cmd)
	require.NoError(t, err)

	assert.True(t, cmd.SilenceErrors)
	assert.True(t, cmd.SilenceUsage)
}

func TestExecuteC_Idempotent(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	opts := &execPlainOpts{}
	var runCount int

	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			runCount++
			return nil
		},
	}

	require.NoError(t, Bind(cmd, opts))

	// First execution
	cmd.SetArgs([]string{"--port", "1111"})
	_, err := ExecuteC(cmd)
	require.NoError(t, err)
	assert.Equal(t, 1111, opts.Port)
	assert.Equal(t, 1, runCount)

	// Second execution on same tree — wrappers should not stack
	cmd.SetArgs([]string{"--port", "2222"})
	_, err = ExecuteC(cmd)
	require.NoError(t, err)
	assert.Equal(t, 2222, opts.Port)
	assert.Equal(t, 2, runCount)
}

func TestExecuteC_NoBoundOptions_StillWorks(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	var ran bool
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			ran = true
			return nil
		},
	}

	_, err := ExecuteC(cmd)
	require.NoError(t, err)
	assert.True(t, ran)
}

func TestExecuteC_AutoSetupUsage(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	root := &cobra.Command{Use: "root"}
	child := &cobra.Command{
		Use: "child",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	root.AddCommand(child)

	// No manual SetupUsage call needed
	root.SetArgs([]string{"child"})
	_, err := ExecuteC(root)
	require.NoError(t, err)

	// Verify annotation was set (proves prepareTree ran)
	assert.Equal(t, "true", root.Annotations[bindPipelineAnnotation])
	assert.Equal(t, "true", child.Annotations[bindPipelineAnnotation])
}

func TestExecuteC_ChildPersistentPreRunE_DoesNotShadowPipeline(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	rootOpts := &execPlainOpts{}
	childOpts := &execChildOpts{}
	var childHookRan bool

	root := &cobra.Command{Use: "root"}
	child := &cobra.Command{
		Use: "child",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	child.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		childHookRan = true
		return nil
	}
	root.AddCommand(child)

	// Bind rootOpts on root (gets defaults), childOpts on child (gets flags)
	require.NoError(t, Bind(root, rootOpts))
	require.NoError(t, Bind(child, childOpts))

	root.SetArgs([]string{"child", "--debug"})
	_, err := ExecuteC(root)
	require.NoError(t, err)

	// Root opts should be unmarshalled with defaults even though child has its own PersistentPreRunE
	assert.Equal(t, 3000, rootOpts.Port, "root bound opts should be unmarshalled with defaults")
	assert.True(t, childOpts.Debug, "child bound opts should be unmarshalled")
	assert.True(t, childHookRan, "child's original PersistentPreRunE should still run")
}

func TestExecuteC_ErrorInUnmarshal_Propagates(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	// Bind an opts struct, then corrupt the scope to force an unmarshal error.
	opts := &execPlainOpts{}
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}

	require.NoError(t, Bind(cmd, opts))

	// Manually add a non-struct to bound options to trigger unmarshal error
	internalscope.Get(cmd).AddBoundOptions("not-a-struct-pointer")

	cmd.SetArgs([]string{})
	_, err := ExecuteC(cmd)
	require.Error(t, err, "unmarshal of invalid bound option should propagate error")
}

func TestExecuteC_ErrorInUserHook_Propagates(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	opts := &execPlainOpts{}
	hookErr := fmt.Errorf("user hook failed")

	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		return hookErr
	}

	require.NoError(t, Bind(cmd, opts))

	cmd.SetArgs([]string{})
	_, err := ExecuteC(cmd)
	require.Error(t, err)
	assert.ErrorIs(t, err, hookErr)
}
