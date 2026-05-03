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

	ok := InferDecodeHooks(cmd, "durations", reflect.TypeFor[[]time.Duration]())
	require.True(t, ok)

	flag := cmd.Flags().Lookup("durations")
	require.NotNil(t, flag)
	assert.Equal(t, []string{"StringToDurationSliceHookFunc"}, flag.Annotations[FlagDecodeHookAnnotation])

	// Unregistered type returns false.
	type missingType struct{}
	found := InferDecodeHooks(cmd, "durations", reflect.TypeFor[missingType]())
	assert.False(t, found)
}

func TestInferDecodeHooks_PanicsOnUnknownFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	// Don't define a "durations" flag; SetAnnotation will panic on unknown flag.

	assert.PanicsWithValue(t,
		`structcli: SetAnnotation on just-registered flag "durations": no such flag -durations`,
		func() { InferDecodeHooks(cmd, "durations", reflect.TypeFor[[]time.Duration]()) },
	)
}

func TestConvertMapInputErrors(t *testing.T) {
	_, err := convertMapInput([]string{"nope"}, convertToString)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid map source type")

	_, err = convertMapInput(map[int]string{1: "value"}, convertToString)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid map key type")
}

func TestConvertToInt(t *testing.T) {
	cases := []struct {
		name  string
		input any
		want  int
	}{
		{name: "string", input: "42", want: 42},
		{name: "int", input: int(7), want: 7},
		{name: "int8", input: int8(8), want: 8},
		{name: "int16", input: int16(9), want: 9},
		{name: "int32", input: int32(10), want: 10},
		{name: "int64", input: int64(11), want: 11},
		{name: "uint", input: uint(12), want: 12},
		{name: "uint8", input: uint8(13), want: 13},
		{name: "uint16", input: uint16(16), want: 16},
		{name: "uint32", input: uint32(17), want: 17},
		{name: "uint64", input: uint64(18), want: 18},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := convertToInt(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	_, err := convertToInt(true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type bool")
}

func TestConvertToInt64(t *testing.T) {
	cases := []struct {
		name  string
		input any
		want  int64
	}{
		{name: "string", input: "42", want: 42},
		{name: "int", input: int(7), want: 7},
		{name: "int8", input: int8(8), want: 8},
		{name: "int16", input: int16(9), want: 9},
		{name: "int32", input: int32(8), want: 8},
		{name: "int64", input: int64(10), want: 10},
		{name: "uint", input: uint(11), want: 11},
		{name: "uint8", input: uint8(12), want: 12},
		{name: "uint16", input: uint16(13), want: 13},
		{name: "uint32", input: uint32(16), want: 16},
		{name: "uint64", input: uint64(17), want: 17},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := convertToInt64(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	_, err := convertToInt64(uint64(^uint64(0)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "overflow")

	_, err = convertToInt64(true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type bool")
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

func TestStringToStringMapHookFunc_ParsesBracketedFlagValue(t *testing.T) {
	out, err := execDecodeHook(
		t,
		StringToStringMapHookFunc(),
		"[env=prod,team=platform]",
		reflect.TypeOf(map[string]string(nil)),
	)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"env": "prod", "team": "platform"}, out)
}

func TestStringToIntMapHookFunc_FromMap(t *testing.T) {
	out, err := execDecodeHook(
		t,
		StringToIntMapHookFunc(),
		map[string]any{"cpu": "8", "memory": 16, "replicas": uint16(2)},
		reflect.TypeOf(map[string]int(nil)),
	)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"cpu": 8, "memory": 16, "replicas": 2}, out)
}

func TestStringToInt64MapHookFunc_FromMap(t *testing.T) {
	out, err := execDecodeHook(
		t,
		StringToInt64MapHookFunc(),
		map[string]any{"ok": "10", "fail": int64(3), "skip": uint32(1)},
		reflect.TypeOf(map[string]int64(nil)),
	)
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{"ok": 10, "fail": 3, "skip": 1}, out)
}

func TestStringToInt64MapHookFunc_InvalidMapValue(t *testing.T) {
	_, err := execDecodeHook(
		t,
		StringToInt64MapHookFunc(),
		map[string]any{"ok": "nope"},
		reflect.TypeOf(map[string]int64(nil)),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid map value for key "ok"`)
}

func TestStoreDecodeHookFuncDirect_WrapperBehavior(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("mode", "", "mode")

	StoreDecodeHookFuncDirect(
		cmd,
		"mode",
		func(input any) (any, error) {
			return strings.ToUpper(input.(string)), nil
		},
		reflect.TypeOf(""),
	)

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

func TestStoreDecodeHookFuncDirect_WrapperPropagatesErrors(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("mode", "", "mode")

	expectedErr := errors.New("boom")
	StoreDecodeHookFuncDirect(
		cmd,
		"mode",
		func(input any) (any, error) {
			return nil, expectedErr
		},
		reflect.TypeOf(""),
	)

	flag := cmd.Flags().Lookup("mode")
	require.NotNil(t, flag)
	hook, exists := internalscope.Get(cmd).GetCustomDecodeHook(flag.Annotations[FlagDecodeHookAnnotation][0])
	require.True(t, exists)
	hookFunc := hook.(func(reflect.Type, reflect.Type, any) (any, error))

	_, err := hookFunc(reflect.TypeOf(""), reflect.TypeOf(""), "dev")
	require.ErrorIs(t, err, expectedErr)
}

type testEnv string

const (
	testEnvDev     testEnv = "dev"
	testEnvStaging testEnv = "staging"
	testEnvProd    testEnv = "prod"
)

func testEnvValues() map[testEnv][]string {
	return map[testEnv][]string{
		testEnvDev:     {"dev", "development"},
		testEnvStaging: {"staging", "stage"},
		testEnvProd:    {"prod", "production"},
	}
}

func TestStringToEnumHookFunc_ValidCanonical(t *testing.T) {
	hook := StringToEnumHookFunc(testEnvValues())

	result, err := execDecodeHook(t, hook, "dev", reflect.TypeFor[testEnv]())
	require.NoError(t, err)
	assert.Equal(t, testEnvDev, result)
}

func TestStringToEnumHookFunc_ValidAlias(t *testing.T) {
	hook := StringToEnumHookFunc(testEnvValues())

	result, err := execDecodeHook(t, hook, "development", reflect.TypeFor[testEnv]())
	require.NoError(t, err)
	assert.Equal(t, testEnvDev, result)
}

func TestStringToEnumHookFunc_CaseInsensitive(t *testing.T) {
	hook := StringToEnumHookFunc(testEnvValues())

	result, err := execDecodeHook(t, hook, "STAGING", reflect.TypeFor[testEnv]())
	require.NoError(t, err)
	assert.Equal(t, testEnvStaging, result)
}

func TestStringToEnumHookFunc_InvalidValue(t *testing.T) {
	hook := StringToEnumHookFunc(testEnvValues())

	_, err := execDecodeHook(t, hook, "invalid", reflect.TypeFor[testEnv]())
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid value "invalid"`)
}

func TestStringToEnumHookFunc_WrongTargetType(t *testing.T) {
	hook := StringToEnumHookFunc(testEnvValues())

	// When target type doesn't match, the hook passes through
	result, err := execDecodeHook(t, hook, "dev", reflect.TypeOf(""))
	require.NoError(t, err)
	assert.Equal(t, "dev", result)
}

func TestStringToEnumHookFunc_NonStringSource(t *testing.T) {
	hook := StringToEnumHookFunc(testEnvValues())

	// When source is not a string, the hook passes through
	result, err := execDecodeHook(t, hook, 42, reflect.TypeFor[testEnv]())
	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestStringToEnumHookFunc_AliasCollisionPanics(t *testing.T) {
	assert.Panics(t, func() {
		StringToEnumHookFunc(map[testEnv][]string{
			testEnvDev:     {"dev"},
			testEnvStaging: {"DEV"},
		})
	})
}

func TestRegisterDecodeHook(t *testing.T) {
	snap := SnapshotDecodeRegistries()
	defer RestoreDecodeRegistries(snap)

	typ := reflect.TypeFor[testEnv]()
	hook := StringToEnumHookFunc(testEnvValues())
	RegisterDecodeHook(typ, "StringToTestEnvHookFunc", hook)

	data, ok := DecodeHookRegistry[typ]
	require.True(t, ok)
	assert.Equal(t, "StringToTestEnvHookFunc", data.ann)

	_, ok = AnnotationToDecodeHookRegistry["StringToTestEnvHookFunc"]
	assert.True(t, ok)
}

func TestRegisterDecodeHook_DuplicatePanics(t *testing.T) {
	snap := SnapshotDecodeRegistries()
	defer RestoreDecodeRegistries(snap)

	hook := StringToEnumHookFunc(testEnvValues())
	RegisterDecodeHook(reflect.TypeFor[testEnv](), "StringToTestEnvHookFunc", hook)

	assert.Panics(t, func() {
		RegisterDecodeHook(reflect.TypeFor[testPriority](), "StringToTestEnvHookFunc", hook)
	})
}

type testPriority int

const (
	testPriorityLow    testPriority = 0
	testPriorityMedium testPriority = 1
	testPriorityHigh   testPriority = 2
)

func testPriorityValues() map[testPriority][]string {
	return map[testPriority][]string{
		testPriorityLow:    {"low"},
		testPriorityMedium: {"medium", "med"},
		testPriorityHigh:   {"high", "hi"},
	}
}

func TestStringToIntEnumHookFunc_ValidCanonical(t *testing.T) {
	hook := StringToIntEnumHookFunc(testPriorityValues())

	result, err := execDecodeHook(t, hook, "low", reflect.TypeFor[testPriority]())
	require.NoError(t, err)
	assert.Equal(t, testPriorityLow, result)
}

func TestStringToIntEnumHookFunc_ValidAlias(t *testing.T) {
	hook := StringToIntEnumHookFunc(testPriorityValues())

	result, err := execDecodeHook(t, hook, "med", reflect.TypeFor[testPriority]())
	require.NoError(t, err)
	assert.Equal(t, testPriorityMedium, result)
}

func TestStringToIntEnumHookFunc_CaseInsensitive(t *testing.T) {
	hook := StringToIntEnumHookFunc(testPriorityValues())

	result, err := execDecodeHook(t, hook, "HIGH", reflect.TypeFor[testPriority]())
	require.NoError(t, err)
	assert.Equal(t, testPriorityHigh, result)
}

func TestStringToIntEnumHookFunc_InvalidValue(t *testing.T) {
	hook := StringToIntEnumHookFunc(testPriorityValues())

	_, err := execDecodeHook(t, hook, "critical", reflect.TypeFor[testPriority]())
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid value "critical"`)
}

func TestStringToIntEnumHookFunc_WrongTargetType(t *testing.T) {
	hook := StringToIntEnumHookFunc(testPriorityValues())

	result, err := execDecodeHook(t, hook, "low", reflect.TypeOf(0))
	require.NoError(t, err)
	assert.Equal(t, "low", result)
}

func TestStringToIntEnumHookFunc_AliasCollisionPanics(t *testing.T) {
	assert.Panics(t, func() {
		StringToIntEnumHookFunc(map[testPriority][]string{
			testPriorityLow:    {"low"},
			testPriorityMedium: {"LOW"},
		})
	})
}
