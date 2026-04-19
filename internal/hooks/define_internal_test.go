package internalhooks

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnumHelpText_SortsAndFormats(t *testing.T) {
	levels := map[int8][]string{
		2:  {"warn"},
		-1: {"debug"},
		0:  {"info"},
		4:  {"error"},
	}

	values, descr := enumHelpText(levels, "log level")

	assert.Equal(t, []string{"debug", "info", "warn", "error"}, values)
	assert.Equal(t, "log level {debug,info,warn,error}", descr)
}

func TestEnumHelpText_EmptyMap(t *testing.T) {
	levels := map[int][]string{}

	values, descr := enumHelpText(levels, "empty")

	assert.Empty(t, values)
	assert.Equal(t, "empty {}", descr)
}

func TestEnumHelpText_SingleEntry(t *testing.T) {
	levels := map[int][]string{
		0: {"only"},
	}

	values, descr := enumHelpText(levels, "one")

	assert.Equal(t, []string{"only"}, values)
	assert.Equal(t, "one {only}", descr)
}

type testColor string

const (
	testColorRed   testColor = "red"
	testColorGreen testColor = "green"
	testColorBlue  testColor = "blue"
)

func TestDefineStringEnumHookFunc_CreatesFlag(t *testing.T) {
	values := map[testColor][]string{
		testColorRed:   {"red"},
		testColorGreen: {"green"},
		testColorBlue:  {"blue"},
	}

	hook := DefineStringEnumHookFunc(values)

	var target testColor = testColorRed
	sf := reflect.StructField{Name: "Color", Type: reflect.TypeFor[testColor]()}
	fv := reflect.ValueOf(&target).Elem()

	pflagVal, usage := hook("color", "c", "pick a color", sf, fv)

	require.NotNil(t, pflagVal)
	assert.Contains(t, usage, "{blue,green,red}")
	assert.Contains(t, usage, "pick a color")

	// Set via the returned pflag.Value
	require.NoError(t, pflagVal.Set("green"))
	assert.Equal(t, testColorGreen, target)
}

func TestDefineStringEnumHookFunc_EnumValues(t *testing.T) {
	values := map[testColor][]string{
		testColorRed:   {"red", "r"},
		testColorGreen: {"green"},
		testColorBlue:  {"blue", "b"},
	}

	hook := DefineStringEnumHookFunc(values)

	var target testColor
	sf := reflect.StructField{Name: "Color", Type: reflect.TypeFor[testColor]()}
	fv := reflect.ValueOf(&target).Elem()

	pflagVal, _ := hook("color", "", "color", sf, fv)

	type enumValuer interface {
		EnumValues() []string
	}
	ev, ok := pflagVal.(enumValuer)
	require.True(t, ok, "returned value should implement EnumValuer")
	assert.Equal(t, []string{"blue", "green", "red"}, ev.EnumValues())
}
