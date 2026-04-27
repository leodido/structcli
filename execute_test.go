package structcli

import (
	"context"
	"fmt"
	"testing"

	"github.com/leodido/structcli/config"
	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/afero"
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

func TestExecuteC_RepeatedExecution_PreservesUserHook(t *testing.T) {
	// Regression: the second ExecuteC on the same tree must still replay
	// the user's original PersistentPreRunE. The hook is saved during the
	// first prepareTree wrap and must persist across executions.
	viper.Reset()
	SetEnvPrefix("")

	opts := &execPlainOpts{}
	var hookCount int

	root := &cobra.Command{Use: "root"}
	root.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		hookCount++
		return nil
	}

	child := &cobra.Command{
		Use: "child",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	root.AddCommand(child)
	require.NoError(t, Bind(child, opts))

	// First execution
	root.SetArgs([]string{"child", "--port", "1111"})
	_, err := ExecuteC(root)
	require.NoError(t, err)
	assert.Equal(t, 1, hookCount, "user hook should fire on first execution")

	// Second execution on same tree
	root.SetArgs([]string{"child", "--port", "2222"})
	_, err = ExecuteC(root)
	require.NoError(t, err)
	assert.Equal(t, 2, hookCount, "user hook should fire on second execution")
}

func TestExecuteC_RepeatedExecution_ReloadsConfig(t *testing.T) {
	// Regression: the second ExecuteC must use a fresh configOnce so that
	// config is reloaded. The wrapper closures must not capture a stale
	// sync.Once from the first call.
	viper.Reset()
	SetEnvPrefix("")

	type appOpts struct {
		Host string `flag:"host" default:"localhost"`
	}

	opts := &appOpts{}

	root := &cobra.Command{Use: "app"}
	child := &cobra.Command{
		Use: "run",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	root.AddCommand(child)

	require.NoError(t, Setup(root, WithAppName("app"), WithConfig(config.Options{})))
	require.NoError(t, Bind(child, opts))

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/etc/app", 0755))

	// First execution: config sets host=alpha
	require.NoError(t, afero.WriteFile(fs, "/etc/app/config.yaml", []byte("host: alpha"), 0644))
	GetConfigViper(root).SetFs(fs)

	root.SetArgs([]string{"run"})
	_, err := ExecuteC(root)
	require.NoError(t, err)
	assert.Equal(t, "alpha", opts.Host, "first execution should load host=alpha from config")

	// Second execution: config changes to host=beta
	require.NoError(t, afero.WriteFile(fs, "/etc/app/config.yaml", []byte("host: beta"), 0644))
	// Reset the config viper so it re-reads the file.
	internalscope.Get(root).ResetConfigViper()
	GetConfigViper(root).SetFs(fs)

	root.SetArgs([]string{"run"})
	_, err = ExecuteC(root)
	require.NoError(t, err)
	assert.Equal(t, "beta", opts.Host, "second execution should reload config and see host=beta")
}

func TestExecuteC_SharedOpts_AncestorLocalFlagFlowsToDescendant(t *testing.T) {
	// Verifies that a local flag defined on root via Bind is visible in a
	// descendant command when the same opts pointer is bound to both root
	// and the descendant. The bind pipeline should unmarshal using the
	// owner command's viper (root) and deduplicate via the seen map.
	viper.Reset()
	SetEnvPrefix("")

	type SharedFlags struct {
		DryRun bool `flag:"dry" flagdescr:"dry run mode"`
	}

	shared := &SharedFlags{}
	var dryInChild bool

	root := &cobra.Command{
		Use:              "app",
		TraverseChildren: true,
	}
	child := &cobra.Command{
		Use: "sub",
		RunE: func(c *cobra.Command, args []string) error {
			dryInChild = shared.DryRun
			return nil
		},
	}
	root.AddCommand(child)

	require.NoError(t, Bind(root, shared))
	require.NoError(t, Bind(child, shared))

	root.SetArgs([]string{"--dry", "sub"})
	_, err := ExecuteC(root)
	require.NoError(t, err)

	assert.True(t, shared.DryRun, "shared.DryRun should be true from --dry flag")
	assert.True(t, dryInChild, "child RunE should see DryRun=true via shared pointer")
}

func TestExecuteC_SharedOpts_SeenMapPreventsDoubleUnmarshal(t *testing.T) {
	// When the same opts pointer is bound to root and child, the bind
	// pipeline should unmarshal it exactly once (on the owner command).
	// A second unmarshal on the child would use the child's viper which
	// doesn't have root's local flags, resetting the value to default.
	viper.Reset()
	SetEnvPrefix("")

	type CountOpts struct {
		Verbose int `flagtype:"count" flagshort:"v"`
	}

	shared := &CountOpts{}

	root := &cobra.Command{
		Use:              "app",
		TraverseChildren: true,
	}
	child := &cobra.Command{
		Use: "sub",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	root.AddCommand(child)

	require.NoError(t, Bind(root, shared))
	require.NoError(t, Bind(child, shared))

	root.SetArgs([]string{"-vvv", "sub"})
	_, err := ExecuteC(root)
	require.NoError(t, err)

	assert.Equal(t, 3, shared.Verbose, "verbose count should be 3, not reset by child unmarshal")
}

func TestExecuteC_SharedContextInjector_VisibleInDescendant(t *testing.T) {
	// A ContextInjector bound to root should have its context visible in
	// descendant commands even when the bind pipeline unmarshals on root.
	viper.Reset()
	SetEnvPrefix("")

	shared := &execContextOpts{}
	var nameInChild string

	root := &cobra.Command{
		Use:              "app",
		TraverseChildren: true,
	}
	child := &cobra.Command{
		Use: "sub",
		RunE: func(c *cobra.Command, args []string) error {
			if v := c.Context().Value(ctxKey("app-name")); v != nil {
				nameInChild = v.(string)
			}
			return nil
		},
	}
	root.AddCommand(child)

	require.NoError(t, Bind(root, shared))

	root.SetArgs([]string{"--app-name", "test-app", "sub"})
	_, err := ExecuteC(root)
	require.NoError(t, err)

	assert.Equal(t, "test-app", shared.AppName)
	assert.Equal(t, "test-app", nameInChild, "context injected on root should be visible in child")
}

