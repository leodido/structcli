package values

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestEnumStringValue_Set_ValidCanonical(t *testing.T) {
	var target testEnv
	ev := NewEnumString(&target, testEnvValues())

	require.NoError(t, ev.Set("dev"))
	assert.Equal(t, testEnvDev, target)

	require.NoError(t, ev.Set("staging"))
	assert.Equal(t, testEnvStaging, target)

	require.NoError(t, ev.Set("prod"))
	assert.Equal(t, testEnvProd, target)
}

func TestEnumStringValue_Set_ValidAlias(t *testing.T) {
	var target testEnv
	ev := NewEnumString(&target, testEnvValues())

	require.NoError(t, ev.Set("development"))
	assert.Equal(t, testEnvDev, target)

	require.NoError(t, ev.Set("stage"))
	assert.Equal(t, testEnvStaging, target)

	require.NoError(t, ev.Set("production"))
	assert.Equal(t, testEnvProd, target)
}

func TestEnumStringValue_Set_CaseInsensitive(t *testing.T) {
	var target testEnv
	ev := NewEnumString(&target, testEnvValues())

	require.NoError(t, ev.Set("DEV"))
	assert.Equal(t, testEnvDev, target)

	require.NoError(t, ev.Set("Staging"))
	assert.Equal(t, testEnvStaging, target)

	require.NoError(t, ev.Set("PRODUCTION"))
	assert.Equal(t, testEnvProd, target)
}

func TestEnumStringValue_Set_Invalid(t *testing.T) {
	var target testEnv = testEnvDev
	ev := NewEnumString(&target, testEnvValues())

	err := ev.Set("invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid value "invalid"`)
	assert.Contains(t, err.Error(), "allowed:")
	assert.Contains(t, err.Error(), "dev")
	// Target unchanged on error
	assert.Equal(t, testEnvDev, target)
}

func TestEnumStringValue_String(t *testing.T) {
	var target testEnv = testEnvStaging
	ev := NewEnumString(&target, testEnvValues())

	assert.Equal(t, "staging", ev.String())

	ev.Set("prod")
	assert.Equal(t, "prod", ev.String())
}

func TestEnumStringValue_String_NilTarget(t *testing.T) {
	ev := &EnumStringValue[testEnv]{}
	assert.Equal(t, "", ev.String())
}

func TestEnumStringValue_Type(t *testing.T) {
	var target testEnv
	ev := NewEnumString(&target, testEnvValues())
	assert.Equal(t, "string", ev.Type())
}

func TestEnumStringValue_EnumValues(t *testing.T) {
	var target testEnv
	ev := NewEnumString(&target, testEnvValues())

	vals := ev.EnumValues()
	// Canonical names sorted alphabetically
	assert.Equal(t, []string{"dev", "prod", "staging"}, vals)
}

func TestEnumStringValue_AliasCollisionPanics(t *testing.T) {
	var target testEnv
	assert.PanicsWithValue(t,
		`values: alias "dev" (lowercased) maps to both dev and staging`,
		func() {
			NewEnumString(&target, map[testEnv][]string{
				testEnvDev:     {"dev"},
				testEnvStaging: {"DEV"}, // collides after lowercasing
			})
		},
	)
}

func TestEnumStringValue_EmptyNames(t *testing.T) {
	var target testEnv
	// An entry with empty names slice is skipped
	ev := NewEnumString(&target, map[testEnv][]string{
		testEnvDev:  {"dev"},
		testEnvProd: {},
	})

	vals := ev.EnumValues()
	assert.Equal(t, []string{"dev"}, vals)

	err := ev.Set("prod")
	require.Error(t, err, "prod has no registered names")
}
