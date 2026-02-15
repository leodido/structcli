package internalpath

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type pathSample struct {
	Value string
}

func TestGetName(t *testing.T) {
	assert.Equal(t, "value", GetName("value", ""))
	assert.Equal(t, "alias", GetName("value", "alias"))
}

func TestGetFieldName(t *testing.T) {
	sf, ok := reflect.TypeOf(pathSample{}).FieldByName("Value")
	assert.True(t, ok)

	assert.Equal(t, "Value", GetFieldName("", sf))
	assert.Equal(t, "Root.Value", GetFieldName("Root", sf))
}

func TestGetFieldPath(t *testing.T) {
	sf, ok := reflect.TypeOf(pathSample{}).FieldByName("Value")
	assert.True(t, ok)

	assert.Equal(t, "value", GetFieldPath("", sf))
	assert.Equal(t, "root.value", GetFieldPath("root", sf))
}
