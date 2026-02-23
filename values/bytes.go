package values

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

func cloneBytes(in []byte) []byte {
	if in == nil {
		return nil
	}
	out := make([]byte, len(in))
	copy(out, in)
	return out
}

type rawBytesValue struct {
	b *[]byte
}

// NewRawBytes creates a pflag.Value that keeps raw textual bytes semantics.
func NewRawBytes(val []byte, p *[]byte) *rawBytesValue {
	*p = cloneBytes(val)
	return &rawBytesValue{b: p}
}

func (r *rawBytesValue) String() string {
	return string(*r.b)
}

func (r *rawBytesValue) Set(val string) error {
	*r.b = []byte(val)
	return nil
}

func (r *rawBytesValue) Type() string {
	return "bytes"
}

var _ pflag.Value = (*rawBytesValue)(nil)

type hexBytesValue struct {
	b *[]byte
}

// NewHexBytes creates a pflag.Value that parses hex-encoded textual input.
func NewHexBytes(val []byte, p *[]byte) *hexBytesValue {
	*p = cloneBytes(val)
	return &hexBytesValue{b: p}
}

func (h *hexBytesValue) String() string {
	return fmt.Sprintf("%X", []byte(*h.b))
}

func (h *hexBytesValue) Set(val string) error {
	decoded, err := hex.DecodeString(strings.TrimSpace(val))
	if err != nil {
		return err
	}
	*h.b = decoded
	return nil
}

func (h *hexBytesValue) Type() string {
	return "bytesHex"
}

var _ pflag.Value = (*hexBytesValue)(nil)

type base64BytesValue struct {
	b *[]byte
}

// NewBase64Bytes creates a pflag.Value that parses base64-encoded textual input.
func NewBase64Bytes(val []byte, p *[]byte) *base64BytesValue {
	*p = cloneBytes(val)
	return &base64BytesValue{b: p}
}

func (b *base64BytesValue) String() string {
	return base64.StdEncoding.EncodeToString([]byte(*b.b))
}

func (b *base64BytesValue) Set(val string) error {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(val))
	if err != nil {
		return err
	}
	*b.b = decoded
	return nil
}

func (b *base64BytesValue) Type() string {
	return "bytesBase64"
}

var _ pflag.Value = (*base64BytesValue)(nil)
