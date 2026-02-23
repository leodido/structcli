package structcli_test

import (
	"reflect"
	"testing"

	"github.com/leodido/structcli"
	"github.com/stretchr/testify/assert"
)

func TestBytesWrapperTypes_AreDistinctFromPlainBytes(t *testing.T) {
	rawType := reflect.TypeOf([]byte{})
	hexType := reflect.TypeOf(structcli.HexBytes{})
	base64Type := reflect.TypeOf(structcli.Base64Bytes{})

	assert.NotEqual(t, rawType, hexType)
	assert.NotEqual(t, rawType, base64Type)
	assert.NotEqual(t, hexType, base64Type)
}
