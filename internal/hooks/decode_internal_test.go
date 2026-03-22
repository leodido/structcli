package internalhooks

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-viper/mapstructure/v2"
	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execDecodeHook(t testing.TB, hook mapstructure.DecodeHookFunc, input any, targetType reflect.Type) (any, error) {
	t.Helper()

	return mapstructure.DecodeHookExec(hook, reflect.ValueOf(input), reflect.New(targetType).Elem())
}

func TestInferDecodeHooks(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("durations", "", "durations")

	ok := InferDecodeHooks(cmd, "durations", "[]time.Duration")
	require.True(t, ok)

	flag := cmd.Flags().Lookup("durations")
	require.NotNil(t, flag)
	assert.Equal(t, []string{"StringToDurationSliceHookFunc"}, flag.Annotations[FlagDecodeHookAnnotation])

	assert.False(t, InferDecodeHooks(cmd, "durations", "missing.Type"))
}

func TestStringToDurationSliceHookFunc_FromArray(t *testing.T) {
	out, err := execDecodeHook(
		t,
		StringToDurationSliceHookFunc(),
		[]any{
			"5s",
			time.Minute,
			int(3),
			int8(4),
			int16(5),
			int32(6),
			int64(7),
			uint(8),
			uint8(9),
			uint16(10),
			uint32(11),
			uint64(12),
		},
		reflect.TypeOf([]time.Duration(nil)),
	)
	require.NoError(t, err)
	assert.Equal(t, []time.Duration{
		5 * time.Second,
		time.Minute,
		3,
		4,
		5,
		6,
		7,
		8,
		9,
		10,
		11,
		12,
	}, out)
}

func TestStringToDurationSliceHookFunc_InvalidArrayElement(t *testing.T) {
	_, err := execDecodeHook(
		t,
		StringToDurationSliceHookFunc(),
		[]any{"5s", true},
		reflect.TypeOf([]time.Duration(nil)),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid element type bool")
}

func TestStringToBoolSliceHookFunc_FromArray(t *testing.T) {
	out, err := execDecodeHook(
		t,
		StringToBoolSliceHookFunc(),
		[]any{"true", false},
		reflect.TypeOf([]bool(nil)),
	)
	require.NoError(t, err)
	assert.Equal(t, []bool{true, false}, out)
}

func TestStringToBoolSliceHookFunc_InvalidArrayElement(t *testing.T) {
	_, err := execDecodeHook(
		t,
		StringToBoolSliceHookFunc(),
		[]any{"true", 1},
		reflect.TypeOf([]bool(nil)),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid element type int")
}

func TestStringToUintSliceHookFunc_FromArray(t *testing.T) {
	out, err := execDecodeHook(
		t,
		StringToUintSliceHookFunc(),
		[]any{
			"5",
			uint(6),
			uint8(7),
			uint16(8),
			uint32(9),
			uint64(10),
			int(11),
			int8(12),
			int16(13),
			int32(14),
			int64(15),
		},
		reflect.TypeOf([]uint(nil)),
	)
	require.NoError(t, err)
	assert.Equal(t, []uint{5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, out)
}

func TestStringToUintSliceHookFunc_InvalidArrayElement(t *testing.T) {
	_, err := execDecodeHook(
		t,
		StringToUintSliceHookFunc(),
		[]any{"5", false},
		reflect.TypeOf([]uint(nil)),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid element type bool")
}

func TestStoreDecodeHookFunc_WrapperBehavior(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("mode", "", "mode")

	err := StoreDecodeHookFunc(
		cmd,
		"mode",
		reflect.ValueOf(func(input any) (any, error) {
			return strings.ToUpper(input.(string)), nil
		}),
		reflect.TypeOf(""),
	)
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("mode")
	require.NotNil(t, flag)
	annotations := flag.Annotations[FlagDecodeHookAnnotation]
	require.Len(t, annotations, 1)

	hook, exists := internalscope.Get(cmd).GetCustomDecodeHook(annotations[0])
	require.True(t, exists)
	hookFunc := hook.(func(reflect.Type, reflect.Type, any) (any, error))

	out, err := hookFunc(reflect.TypeOf(""), reflect.TypeOf(""), "dev")
	require.NoError(t, err)
	assert.Equal(t, "DEV", out)

	out, err = hookFunc(reflect.TypeOf(""), reflect.TypeOf(int(0)), "dev")
	require.NoError(t, err)
	assert.Equal(t, "dev", out)
}

func TestStoreDecodeHookFunc_WrapperPropagatesErrors(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("mode", "", "mode")

	expectedErr := errors.New("boom")
	err := StoreDecodeHookFunc(
		cmd,
		"mode",
		reflect.ValueOf(func(input any) (any, error) {
			return nil, expectedErr
		}),
		reflect.TypeOf(""),
	)
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("mode")
	require.NotNil(t, flag)
	hook, exists := internalscope.Get(cmd).GetCustomDecodeHook(flag.Annotations[FlagDecodeHookAnnotation][0])
	require.True(t, exists)
	hookFunc := hook.(func(reflect.Type, reflect.Type, any) (any, error))

	_, err = hookFunc(reflect.TypeOf(""), reflect.TypeOf(""), "dev")
	require.ErrorIs(t, err, expectedErr)
}
