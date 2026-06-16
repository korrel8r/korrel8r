// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package json is a drop-in replacement for encoding/json using bytedance/sonic for performance.
// On amd64/arm64, sonic uses JIT compilation for faster JSON processing.
// On other architectures, sonic falls back to encoding/json transparently.
package json

import (
	stdjson "encoding/json"
	"io"

	"github.com/bytedance/sonic"
)

// Type aliases from encoding/json for compatibility.
type (
	RawMessage           = stdjson.RawMessage
	Number               = stdjson.Number
	Marshaler            = stdjson.Marshaler
	Unmarshaler          = stdjson.Unmarshaler
	InvalidUnmarshalError = stdjson.InvalidUnmarshalError
)

var std = sonic.ConfigStd

func Marshal(v any) ([]byte, error)                           { return std.Marshal(v) }
func MarshalIndent(v any, prefix, indent string) ([]byte, error) { return std.MarshalIndent(v, prefix, indent) }
func Unmarshal(data []byte, v any) error                      { return std.Unmarshal(data, v) }
func Valid(data []byte) bool                                  { return std.Valid(data) }

// Encoder writes JSON to an output stream.
type Encoder struct{ sonic.Encoder }

func NewEncoder(w io.Writer) *Encoder { return &Encoder{std.NewEncoder(w)} }

// Decoder reads JSON from an input stream.
type Decoder struct{ sonic.Decoder }

func NewDecoder(r io.Reader) *Decoder { return &Decoder{std.NewDecoder(r)} }
