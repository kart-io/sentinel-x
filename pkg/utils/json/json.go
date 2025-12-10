// Package json provides a high-performance JSON serialization wrapper.
// It automatically uses sonic for supported architectures (amd64/arm64) and
// falls back to standard encoding/json for other platforms.
package json

import (
	stdjson "encoding/json"
	"io"
	"runtime"

	"github.com/bytedance/sonic"
)

var (
	// Marshal encodes v into JSON bytes.
	// Uses sonic on amd64/arm64, otherwise falls back to encoding/json.
	Marshal func(v interface{}) ([]byte, error)

	// Unmarshal decodes JSON bytes into v.
	// Uses sonic on amd64/arm64, otherwise falls back to encoding/json.
	Unmarshal func(data []byte, v interface{}) error

	// NewEncoder creates a new JSON encoder for the writer.
	NewEncoder func(w io.Writer) Encoder

	// NewDecoder creates a new JSON decoder for the reader.
	NewDecoder func(r io.Reader) Decoder

	// usingSonic indicates whether sonic is being used.
	usingSonic bool
)

// Encoder is a JSON encoder interface.
type Encoder interface {
	Encode(v interface{}) error
}

// Decoder is a JSON decoder interface.
type Decoder interface {
	Decode(v interface{}) error
}

func init() {
	// Sonic only supports amd64 and arm64 architectures
	if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
		// Use sonic's default configuration (balances performance and compatibility)
		Marshal = sonic.Marshal
		Unmarshal = sonic.Unmarshal
		NewEncoder = func(w io.Writer) Encoder {
			return sonic.ConfigDefault.NewEncoder(w)
		}
		NewDecoder = func(r io.Reader) Decoder {
			return sonic.ConfigDefault.NewDecoder(r)
		}
		usingSonic = true
	} else {
		// Fallback to standard library for unsupported architectures
		Marshal = stdjson.Marshal
		Unmarshal = stdjson.Unmarshal
		NewEncoder = func(w io.Writer) Encoder {
			return stdjson.NewEncoder(w)
		}
		NewDecoder = func(r io.Reader) Decoder {
			return stdjson.NewDecoder(r)
		}
		usingSonic = false
	}
}

// ConfigFastestMode switches to sonic's fastest mode.
// This mode disables some safety checks for maximum performance.
// Only affects sonic implementation; no-op for standard library fallback.
//
// Use this when:
// - You control the input data and trust it
// - Maximum throughput is critical
// - The data structures are simple and well-tested
//
// DO NOT use this when:
// - Handling untrusted external input
// - Data structures have complex validation requirements
func ConfigFastestMode() {
	if usingSonic {
		api := sonic.ConfigFastest
		Marshal = api.Marshal
		Unmarshal = api.Unmarshal
		NewEncoder = func(w io.Writer) Encoder {
			return api.NewEncoder(w)
		}
		NewDecoder = func(r io.Reader) Decoder {
			return api.NewDecoder(r)
		}
	}
}

// ConfigStandardMode switches to sonic's standard mode.
// This is the default mode with balanced performance and safety.
// Only affects sonic implementation; no-op for standard library fallback.
func ConfigStandardMode() {
	if usingSonic {
		api := sonic.ConfigDefault
		Marshal = api.Marshal
		Unmarshal = api.Unmarshal
		NewEncoder = func(w io.Writer) Encoder {
			return api.NewEncoder(w)
		}
		NewDecoder = func(r io.Reader) Decoder {
			return api.NewDecoder(r)
		}
	}
}

// IsUsingSonic returns true if sonic is being used for JSON operations.
func IsUsingSonic() bool {
	return usingSonic
}
