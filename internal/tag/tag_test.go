package internaltag

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type customInt int

type mandatorySample struct {
	Required string `flagrequired:"true"`
	Optional string
	Invalid  string `flagrequired:"not-a-bool"`
}

func TestIsValidFlagName(t *testing.T) {
	valid := []string{"port", "log-level", "db.url", "a1", "A.B-c"}
	for _, name := range valid {
		assert.True(t, IsValidFlagName(name), name)
	}

	invalid := []string{"", "-lead", "trail-", "two--dashes", "with space", "with_underscore"}
	for _, name := range invalid {
		assert.False(t, IsValidFlagName(name), name)
	}
}

func TestIsStandardType(t *testing.T) {
	assert.True(t, IsStandardType(reflect.TypeOf(int(0))))
	assert.True(t, IsStandardType(reflect.TypeOf(float64(0))))
	assert.False(t, IsStandardType(reflect.TypeOf(customInt(0))))
	assert.False(t, IsStandardType(reflect.TypeOf([]string{})))
}

func TestIsMandatory(t *testing.T) {
	T := reflect.TypeOf(mandatorySample{})

	required, ok := T.FieldByName("Required")
	assert.True(t, ok)
	assert.True(t, IsMandatory(required))

	optional, ok := T.FieldByName("Optional")
	assert.True(t, ok)
	assert.False(t, IsMandatory(optional))

	invalid, ok := T.FieldByName("Invalid")
	assert.True(t, ok)
	assert.False(t, IsMandatory(invalid))
}

func TestParseFlagPresets(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got, err := ParseFlagPresets("")
		assert.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("single", func(t *testing.T) {
		got, err := ParseFlagPresets("logeverything=5")
		assert.NoError(t, err)
		assert.Equal(t, []FlagPreset{{Name: "logeverything", Value: "5"}}, got)
	})

	t.Run("multiple_semicolon", func(t *testing.T) {
		got, err := ParseFlagPresets("logeverything=5;logquiet=0")
		assert.NoError(t, err)
		assert.Equal(t, []FlagPreset{
			{Name: "logeverything", Value: "5"},
			{Name: "logquiet", Value: "0"},
		}, got)
	})

	t.Run("multiple_comma", func(t *testing.T) {
		got, err := ParseFlagPresets("a=1,b=2")
		assert.NoError(t, err)
		assert.Equal(t, []FlagPreset{
			{Name: "a", Value: "1"},
			{Name: "b", Value: "2"},
		}, got)
	})

	t.Run("value_can_contain_equals", func(t *testing.T) {
		got, err := ParseFlagPresets("token=foo=bar")
		assert.NoError(t, err)
		assert.Equal(t, []FlagPreset{{Name: "token", Value: "foo=bar"}}, got)
	})

	t.Run("value_can_be_empty", func(t *testing.T) {
		got, err := ParseFlagPresets("clear=")
		assert.NoError(t, err)
		assert.Equal(t, []FlagPreset{{Name: "clear", Value: ""}}, got)
	})

	t.Run("invalid_missing_equals", func(t *testing.T) {
		_, err := ParseFlagPresets("logeverything")
		assert.Error(t, err)
	})

	t.Run("invalid_name", func(t *testing.T) {
		_, err := ParseFlagPresets("bad_name=1")
		assert.Error(t, err)
	})

	t.Run("duplicate_names", func(t *testing.T) {
		_, err := ParseFlagPresets("same=1;same=2")
		assert.Error(t, err)
	})

	t.Run("empty_entries_not_allowed", func(t *testing.T) {
		_, err := ParseFlagPresets("a=1;;b=2")
		assert.Error(t, err)
	})
}
