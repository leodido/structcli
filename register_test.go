package structcli

import (
	"reflect"
	"testing"

	internalhooks "github.com/leodido/structcli/internal/hooks"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCustomType struct{ val string }

func TestRegisterType_Success(t *testing.T) {
	typ := reflect.TypeFor[testCustomType]()
	snap := internalhooks.SnapshotDecodeRegistries()
	defer func() {
		internalhooks.RestoreDecodeRegistries(snap)
		delete(internalhooks.DefineHookRegistry, typ)
	}()

	RegisterType(TypeHooks[testCustomType]{
		Define: func(name, descr string, sf reflect.StructField, fv reflect.Value) (pflag.Value, string) {
			return nil, descr
		},
		Decode: func(input any) (any, error) {
			return testCustomType{val: input.(string)}, nil
		},
	})

	// Verify define hook registered
	defineHook, ok := internalhooks.DefineHookRegistry[typ]
	require.True(t, ok, "define hook should be registered")
	assert.NotNil(t, defineHook)

	// Verify decode hook registered
	_, ok = internalhooks.DecodeHookRegistry[typ]
	assert.True(t, ok, "decode hook should be registered")
}

func TestRegisterType_PanicsOnNilDefine(t *testing.T) {
	assert.PanicsWithValue(t,
		"structcli: RegisterType[structcli.testCustomType]: Define hook must not be nil",
		func() {
			RegisterType(TypeHooks[testCustomType]{
				Define: nil,
				Decode: func(input any) (any, error) { return input, nil },
			})
		},
	)
}

func TestRegisterType_PanicsOnNilDecode(t *testing.T) {
	assert.PanicsWithValue(t,
		"structcli: RegisterType[structcli.testCustomType]: Decode hook must not be nil",
		func() {
			RegisterType(TypeHooks[testCustomType]{
				Define: func(name, descr string, sf reflect.StructField, fv reflect.Value) (pflag.Value, string) {
					return nil, descr
				},
				Decode: nil,
			})
		},
	)
}

func TestRegisterType_PanicsOnDuplicate(t *testing.T) {
	typ := reflect.TypeFor[testCustomType]()
	snap := internalhooks.SnapshotDecodeRegistries()
	defer func() {
		internalhooks.RestoreDecodeRegistries(snap)
		delete(internalhooks.DefineHookRegistry, typ)
	}()

	hooks := TypeHooks[testCustomType]{
		Define: func(name, descr string, sf reflect.StructField, fv reflect.Value) (pflag.Value, string) {
			return nil, descr
		},
		Decode: func(input any) (any, error) { return input, nil },
	}

	RegisterType(hooks)

	assert.Panics(t, func() {
		RegisterType(hooks)
	}, "duplicate registration should panic")
}
