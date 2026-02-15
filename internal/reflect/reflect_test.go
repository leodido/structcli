package internalreflect

import (
	"errors"
	"reflect"
	"testing"

	structclierrors "github.com/leodido/structcli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type reflectSample struct {
	Name string
}

func TestGetValue(t *testing.T) {
	val := GetValue(reflectSample{Name: "x"})
	require.True(t, val.IsValid())
	assert.Equal(t, reflect.Struct, val.Kind())
	assert.Equal(t, "x", val.FieldByName("Name").String())

	ptrVal := GetValue(&reflectSample{Name: "y"})
	require.True(t, ptrVal.IsValid())
	assert.Equal(t, reflect.Struct, ptrVal.Kind())
	assert.Equal(t, "y", ptrVal.FieldByName("Name").String())

	invalid := GetValue(nil)
	assert.False(t, invalid.IsValid())
}

func TestGetValuePtr(t *testing.T) {
	val := GetValuePtr(reflectSample{Name: "x"})
	require.True(t, val.IsValid())
	assert.Equal(t, reflect.Ptr, val.Kind())
	assert.Equal(t, reflect.TypeOf(reflectSample{}), val.Elem().Type())

	var nilPtr *reflectSample
	got := GetValuePtr(nilPtr)
	require.True(t, got.IsValid())
	assert.Equal(t, reflect.Ptr, got.Kind())
	assert.False(t, got.IsNil())
	assert.Equal(t, reflect.TypeOf(reflectSample{}), got.Elem().Type())
}

func TestGetStructPtr(t *testing.T) {
	nonAddr := reflect.ValueOf(reflectSample{Name: "x"})
	ptr := GetStructPtr(nonAddr)
	require.True(t, ptr.IsValid())
	assert.Equal(t, reflect.Ptr, ptr.Kind())
	assert.Equal(t, "x", ptr.Elem().FieldByName("Name").String())

	addr := reflect.ValueOf(&reflectSample{Name: "y"}).Elem()
	addrPtr := GetStructPtr(addr)
	require.True(t, addrPtr.IsValid())
	assert.Equal(t, reflect.Ptr, addrPtr.Kind())
	assert.Equal(t, "y", addrPtr.Elem().FieldByName("Name").String())

	var nilPtr *reflectSample
	nilPtrVal := GetStructPtr(reflect.ValueOf(nilPtr))
	require.True(t, nilPtrVal.IsValid())
	assert.Equal(t, reflect.Ptr, nilPtrVal.Kind())
	assert.False(t, nilPtrVal.IsNil())

	assert.False(t, GetStructPtr(reflect.Value{}).IsValid())
}

func TestGetValidValue(t *testing.T) {
	_, err := GetValidValue(nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, structclierrors.ErrInputValue))

	var nilPtr *reflectSample
	val, err := GetValidValue(nilPtr)
	require.NoError(t, err)
	require.True(t, val.IsValid())
	assert.Equal(t, reflect.Struct, val.Kind())
	assert.Equal(t, "", val.FieldByName("Name").String())
}

func TestSignature(t *testing.T) {
	fx := func(string, int) (bool, error) { return true, nil }
	assert.Equal(t, "func(string, int) (bool, error)", Signature(fx))
	assert.Equal(t, "<not a function>", Signature(123))
}
