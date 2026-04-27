package structcli

import (
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

	// Should succeed — Validatable and Transformable don't require Attach
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
