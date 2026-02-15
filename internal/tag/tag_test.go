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
