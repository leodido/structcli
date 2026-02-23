package values

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRawBytesValue(t *testing.T) {
	t.Run("NewCopiesInitialValue", func(t *testing.T) {
		initial := []byte("token")
		var target []byte

		v := NewRawBytes(initial, &target)
		initial[0] = 'x'

		assert.Equal(t, []byte("token"), target)
		assert.Equal(t, "token", v.String())
	})

	t.Run("SetStoresRawBytes", func(t *testing.T) {
		var target []byte
		v := NewRawBytes(nil, &target)

		err := v.Set("abc,123")
		require.NoError(t, err)
		assert.Equal(t, []byte("abc,123"), target)
		assert.Equal(t, "abc,123", v.String())
		assert.Equal(t, "bytes", v.Type())
	})
}

func TestHexBytesValue(t *testing.T) {
	t.Run("NewCopiesInitialValue", func(t *testing.T) {
		initial := []byte{0xDE, 0xAD}
		var target []byte

		v := NewHexBytes(initial, &target)
		initial[0] = 0x00

		assert.Equal(t, []byte{0xDE, 0xAD}, target)
		assert.Equal(t, "DEAD", v.String())
	})

	t.Run("SetValidHex", func(t *testing.T) {
		var target []byte
		v := NewHexBytes(nil, &target)

		err := v.Set("68656c6c6f")
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), target)
		assert.Equal(t, "68656C6C6F", v.String())
		assert.Equal(t, "bytesHex", v.Type())
	})

	t.Run("SetInvalidHexKeepsPreviousValue", func(t *testing.T) {
		target := []byte("stable")
		v := NewHexBytes(target, &target)

		err := v.Set("xyz")
		require.Error(t, err)
		assert.Equal(t, []byte("stable"), target)
	})
}

func TestBase64BytesValue(t *testing.T) {
	t.Run("NewCopiesInitialValue", func(t *testing.T) {
		initial := []byte("hello")
		var target []byte

		v := NewBase64Bytes(initial, &target)
		initial[0] = 'x'

		assert.Equal(t, []byte("hello"), target)
		assert.Equal(t, "aGVsbG8=", v.String())
	})

	t.Run("SetValidBase64", func(t *testing.T) {
		var target []byte
		v := NewBase64Bytes(nil, &target)

		err := v.Set("aGVsbG8=")
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), target)
		assert.Equal(t, "aGVsbG8=", v.String())
		assert.Equal(t, "bytesBase64", v.Type())
	})

	t.Run("SetInvalidBase64KeepsPreviousValue", func(t *testing.T) {
		target := []byte("stable")
		v := NewBase64Bytes(target, &target)

		err := v.Set("@@@")
		require.Error(t, err)
		assert.Equal(t, []byte("stable"), target)
	})
}
