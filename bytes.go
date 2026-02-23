package structcli

// Byte-related type contract for structcli:
//   - []byte keeps raw textual bytes semantics.
//   - HexBytes opts into hex-encoded textual input.
//   - Base64Bytes opts into base64-encoded textual input.
//
// These wrapper types are intentionally distinct from []byte so Define/Unmarshal
// integrations can infer the intended parsing strategy from the Go type.

// HexBytes represents binary data provided as hex-encoded textual input.
type HexBytes []byte

// Base64Bytes represents binary data provided as base64-encoded textual input.
type Base64Bytes []byte
