package structcli

import (
	"bytes"
	"context"
	"testing"

	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Plain struct (no Attach) ---

type bindPlainOpts struct {
	Port int    `flag:"port" flagshort:"p" flagdescr:"server port" default:"3000"`
	Host string `flag:"host" flagdescr:"server host" default:"localhost"`
}

// --- Options implementor (has Attach) ---

type bindAttachOpts struct {
	Verbose bool `flag:"verbose" flagshort:"v" flagdescr:"enable verbose output"`
}

func (o *bindAttachOpts) Attach(c *cobra.Command) error {
	return Define(c, o)
}

// --- Standalone capability interfaces (no Attach) ---

type bindValidatableOpts struct {
	Name string `flag:"name" flagdescr:"user name" default:"world"`
}

func (o *bindValidatableOpts) Validate(context.Context) []error {
	if o.Name == "" {
		return []error{assert.AnError}
	}

	return nil
}

func (o *bindValidatableOpts) Transform(context.Context) error {
	return nil
}

// --- Tests ---

func TestBind_NilCommand(t *testing.T) {
	err := Bind(nil, &bindPlainOpts{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command must not be nil")
}

func TestBind_NilOpts(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	err := Bind(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opts must not be nil")
}

func TestBind_NonStructPointer(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}

	err := Bind(cmd, "not a struct")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "struct pointer")

	num := 42
	err = Bind(cmd, &num)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "struct pointer")
}

func TestBind_PlainStruct_DefinesFlags(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	cmd := &cobra.Command{Use: "test"}
	opts := &bindPlainOpts{}

	err := Bind(cmd, opts)
	require.NoError(t, err)

	// Flags should be registered
	portFlag := cmd.Flags().Lookup("port")
	require.NotNil(t, portFlag, "port flag should be defined")
	assert.Equal(t, "p", portFlag.Shorthand)
	assert.Equal(t, "3000", portFlag.DefValue)

	hostFlag := cmd.Flags().Lookup("host")
	require.NotNil(t, hostFlag, "host flag should be defined")
	assert.Equal(t, "localhost", hostFlag.DefValue)
}

func TestBind_OptionsImplementor_CallsAttach(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	cmd := &cobra.Command{Use: "test"}
	opts := &bindAttachOpts{}

	err := Bind(cmd, opts)
	require.NoError(t, err)

	// Flags should be registered via Attach → Define
	verboseFlag := cmd.Flags().Lookup("verbose")
	require.NotNil(t, verboseFlag, "verbose flag should be defined")
	assert.Equal(t, "v", verboseFlag.Shorthand)
}

func TestBind_RegistersInScope(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	cmd := &cobra.Command{Use: "test"}
	opts := &bindPlainOpts{}

	err := Bind(cmd, opts)
	require.NoError(t, err)

	scope := internalscope.Get(cmd)
	bound := scope.BoundOptions()
	require.Len(t, bound, 1)
	assert.Same(t, opts, bound[0].(*bindPlainOpts))
}

func TestBind_MultipleCalls_PreservesOrder(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	cmd := &cobra.Command{Use: "test"}
	plain := &bindPlainOpts{}
	attach := &bindAttachOpts{}

	require.NoError(t, Bind(cmd, plain))
	require.NoError(t, Bind(cmd, attach))

	scope := internalscope.Get(cmd)
	bound := scope.BoundOptions()
	require.Len(t, bound, 2)
	assert.Same(t, plain, bound[0].(*bindPlainOpts), "first Bind should be first in list")
	assert.Same(t, attach, bound[1].(*bindAttachOpts), "second Bind should be second in list")
}

func TestBind_PlainStruct_WithCapabilityInterfaces(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	cmd := &cobra.Command{Use: "test"}
	opts := &bindValidatableOpts{}

	// Should succeed. Validatable and Transformable don't require Attach.
	err := Bind(cmd, opts)
	require.NoError(t, err)

	nameFlag := cmd.Flags().Lookup("name")
	require.NotNil(t, nameFlag, "name flag should be defined")

	// Verify it's registered in scope
	scope := internalscope.Get(cmd)
	bound := scope.BoundOptions()
	require.Len(t, bound, 1)

	// Verify it satisfies standalone interfaces but not Options
	_, isValidatable := bound[0].(Validatable)
	_, isTransformable := bound[0].(Transformable)
	_, isOptions := bound[0].(Options)
	assert.True(t, isValidatable)
	assert.True(t, isTransformable)
	assert.False(t, isOptions)
}

func TestBind_PlainStruct_FlagsWorkWithParsing(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	cmd := &cobra.Command{Use: "test"}
	opts := &bindPlainOpts{}

	require.NoError(t, Bind(cmd, opts))

	// Simulate flag parsing
	cmd.SetArgs([]string{"--port", "8080", "--host", "0.0.0.0"})
	require.NoError(t, cmd.ParseFlags([]string{"--port", "8080", "--host", "0.0.0.0"}))

	// Flags should have the parsed values
	portFlag := cmd.Flags().Lookup("port")
	assert.Equal(t, "8080", portFlag.Value.String())

	hostFlag := cmd.Flags().Lookup("host")
	assert.Equal(t, "0.0.0.0", hostFlag.Value.String())
}

func TestBind_WarnsWhenExecuteCNotUsed(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	var stderr bytes.Buffer
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.SetErr(&stderr)

	opts := &bindPlainOpts{}
	require.NoError(t, Bind(cmd, opts))

	// Use cmd.Execute() directly, not structcli.ExecuteC.
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	assert.Contains(t, stderr.String(), "ExecuteC")
	assert.Contains(t, stderr.String(), "auto-unmarshalled")
}

func TestBind_NoWarningWhenExecuteCUsed(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	var stderr bytes.Buffer
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.SetErr(&stderr)

	opts := &bindPlainOpts{}
	require.NoError(t, Bind(cmd, opts))

	cmd.SetArgs([]string{})
	_, err := ExecuteC(cmd)
	require.NoError(t, err)

	assert.Empty(t, stderr.String(), "no warning should be printed when ExecuteC is used")
}

func TestBind_NoWarningWhenNoBind(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	var stderr bytes.Buffer
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.SetErr(&stderr)

	// No Bind call; just execute directly.
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	assert.Empty(t, stderr.String(), "no warning should be printed when Bind was never called")
}

func TestBind_Warning_IndependentCommandTrees(t *testing.T) {
	// Two independent command trees should each get their own warning
	// hook. The per-tree PersistentPreRunE approach (no global state)
	// means each tree is self-contained.
	viper.Reset()
	SetEnvPrefix("")

	// Tree 1: uses cmd.Execute(), should warn.
	var stderr1 bytes.Buffer
	cmd1 := &cobra.Command{
		Use: "tree1",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd1.SetErr(&stderr1)
	require.NoError(t, Bind(cmd1, &bindPlainOpts{}))

	cmd1.SetArgs([]string{})
	require.NoError(t, cmd1.Execute())
	assert.Contains(t, stderr1.String(), "ExecuteC", "tree1 should warn when cmd.Execute() is used")

	// Tree 2: uses ExecuteC, should not warn.
	var stderr2 bytes.Buffer
	cmd2 := &cobra.Command{
		Use: "tree2",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd2.SetErr(&stderr2)
	require.NoError(t, Bind(cmd2, &bindPlainOpts{}))

	cmd2.SetArgs([]string{})
	_, err := ExecuteC(cmd2)
	require.NoError(t, err)
	assert.Empty(t, stderr2.String(), "tree2 should not warn when ExecuteC is used")
}

// ExecuteOrExit delegates to ExecuteC, so the executeCActiveAnnotation is
// set before the PersistentPreRunE fires. No separate test needed; the
// annotation path is covered by TestBind_NoWarningWhenExecuteCUsed.
// Testing ExecuteOrExit directly would require mocking os.Exit.

func TestBind_Warning_NoFalsePositiveAfterExecuteC(t *testing.T) {
	// After ExecuteC installs the pipeline wrapper, a subsequent
	// cmd.Execute() on the same tree should NOT warn because the pipeline
	// is already installed and auto-unmarshal works.
	viper.Reset()
	SetEnvPrefix("")

	var stderr bytes.Buffer
	opts := &bindPlainOpts{}
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.SetErr(&stderr)
	require.NoError(t, Bind(cmd, opts))

	// First call via ExecuteC installs pipeline wrapper.
	cmd.SetArgs([]string{"--port", "8080"})
	_, err := ExecuteC(cmd)
	require.NoError(t, err)
	assert.Empty(t, stderr.String(), "no warning expected from ExecuteC")
	assert.Equal(t, 8080, opts.Port)

	// Second call via cmd.Execute(). Pipeline wrapper persists,
	// auto-unmarshal still works, no warning should fire.
	stderr.Reset()
	cmd.SetArgs([]string{"--port", "7070"})
	require.NoError(t, cmd.Execute())
	assert.Empty(t, stderr.String(), "no warning when pipeline wrapper is already installed")
	assert.Equal(t, 7070, opts.Port, "pipeline should still unmarshal via the persisted wrapper")
}

func TestBind_Warning_ShadowedByChildPersistentPreRunE(t *testing.T) {
	// When a child command has its own PersistentPreRunE, Cobra picks it
	// and never reaches root's warning hook. This is a known limitation.
	viper.Reset()
	SetEnvPrefix("")

	var stderr bytes.Buffer
	root := &cobra.Command{Use: "app"}
	child := &cobra.Command{
		Use: "sub",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error { return nil },
	}
	root.AddCommand(child)
	root.SetErr(&stderr)

	require.NoError(t, Bind(child, &bindPlainOpts{}))

	// cmd.Execute(): warning is shadowed by child's PersistentPreRunE.
	root.SetArgs([]string{"sub"})
	require.NoError(t, root.Execute())
	assert.Empty(t, stderr.String(), "warning is shadowed when child has PersistentPreRunE (known limitation)")
}

func TestBind_Warning_OverwrittenByUserHook(t *testing.T) {
	// If the user sets root.PersistentPreRunE after Bind, the warning
	// hook is silently replaced.
	viper.Reset()
	SetEnvPrefix("")

	var stderr bytes.Buffer
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(c *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.SetErr(&stderr)
	require.NoError(t, Bind(cmd, &bindPlainOpts{}))

	// Overwrite root's PersistentPreRunE after Bind.
	cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		return nil
	}

	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())
	assert.Empty(t, stderr.String(), "warning is lost when user overwrites PersistentPreRunE after Bind")
}

func TestBind_Warning_BindOnSubcommand(t *testing.T) {
	// Bind on a subcommand (not root) should still install the warning
	// hook on root.
	viper.Reset()
	SetEnvPrefix("")

	var stderr bytes.Buffer
	root := &cobra.Command{Use: "app"}
	child := &cobra.Command{
		Use:  "sub",
		RunE: func(c *cobra.Command, args []string) error { return nil },
	}
	root.AddCommand(child)
	root.SetErr(&stderr)

	require.NoError(t, Bind(child, &bindPlainOpts{}))

	// cmd.Execute(): warning should fire from root's hook.
	root.SetArgs([]string{"sub"})
	require.NoError(t, root.Execute())
	assert.Contains(t, stderr.String(), "ExecuteC", "warning should fire even when Bind was called on a subcommand")
}

func TestBind_ScopeIsolation_DifferentCommands(t *testing.T) {
	viper.Reset()
	SetEnvPrefix("")

	root := &cobra.Command{Use: "root"}
	child := &cobra.Command{Use: "child"}
	root.AddCommand(child)

	rootOpts := &bindPlainOpts{}
	childOpts := &bindAttachOpts{}

	require.NoError(t, Bind(root, rootOpts))
	require.NoError(t, Bind(child, childOpts))

	rootBound := internalscope.Get(root).BoundOptions()
	childBound := internalscope.Get(child).BoundOptions()

	require.Len(t, rootBound, 1)
	require.Len(t, childBound, 1)
	assert.Same(t, rootOpts, rootBound[0].(*bindPlainOpts))
	assert.Same(t, childOpts, childBound[0].(*bindAttachOpts))
}
