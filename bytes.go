package structcli

import (
	"reflect"

	internalhooks "github.com/leodido/structcli/internal/hooks"
)

// Byte-related type contract:
//   - []byte keeps raw textual bytes semantics.
//   - Hex opts into hex-encoded textual input.
//   - Base64 opts into base64-encoded textual input.
//
// These wrapper types are intentionally distinct from []byte so Define/Unmarshal
// integrations can infer the intended parsing strategy from the Go type.

// Hex represents binary data provided as hex-encoded textual input.
type Hex []byte

// Base64 represents binary data provided as base64-encoded textual input.
type Base64 []byte

func init() {
	hexType := reflect.TypeFor[Hex]()
	internalhooks.DefineHookRegistry[hexType] = internalhooks.DefineHexBytesHookFunc()
	internalhooks.RegisterDecodeHook(hexType, "StringToHexHookFunc",
		internalhooks.StringToNamedBytesHookFunc(hexType, internalhooks.DecodeHexBytes))

	b64Type := reflect.TypeFor[Base64]()
	internalhooks.DefineHookRegistry[b64Type] = internalhooks.DefineBase64BytesHookFunc()
	internalhooks.RegisterDecodeHook(b64Type, "StringToBase64HookFunc",
		internalhooks.StringToNamedBytesHookFunc(b64Type, internalhooks.DecodeBase64Bytes))
}
