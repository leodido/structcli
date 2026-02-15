package internalconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type validateDatabaseOptions struct {
	URL string `flag:"db-url"`
}

type validateServiceOptions struct {
	Port     int `flag:"port"`
	Database validateDatabaseOptions
}

func TestValidateKeys_EmptyMap(t *testing.T) {
	err := ValidateKeys(map[string]any{}, &validateServiceOptions{})
	require.NoError(t, err)
}

func TestValidateKeys_NilTarget(t *testing.T) {
	err := ValidateKeys(map[string]any{"port": 8080}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil options target")
}

func TestValidateKeys_NonStructTarget(t *testing.T) {
	var target string
	err := ValidateKeys(map[string]any{"port": 8080}, &target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a struct")
}

func TestValidateKeys_UnknownTopLevelKey(t *testing.T) {
	err := ValidateKeys(map[string]any{
		"port":  8080,
		"extra": "nope",
	}, &validateServiceOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown config keys")
	assert.Contains(t, err.Error(), "extra")
}

func TestValidateKeys_UnknownNestedKey(t *testing.T) {
	err := ValidateKeys(map[string]any{
		"database": map[string]any{
			"url":   "postgres://localhost/db",
			"extra": "nope",
		},
	}, &validateServiceOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown config keys")
	assert.Contains(t, err.Error(), "database.extra")
}

func TestValidateKeys_FlattenedAliasKey(t *testing.T) {
	err := ValidateKeys(
		map[string]any{
			"db-url": "postgres://localhost/db",
		},
		&validateServiceOptions{},
		KeyRemappingHook(map[string]string{"db-url": "database.url"}, map[string]string{}),
	)
	require.NoError(t, err)
}

func TestValidateKeys_AcceptsNestedFieldNameKey(t *testing.T) {
	err := ValidateKeys(map[string]any{
		"database": map[string]any{
			"url": "postgres://localhost/db",
		},
	}, &validateServiceOptions{})
	require.NoError(t, err)
}
