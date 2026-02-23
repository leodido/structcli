package structcli

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
