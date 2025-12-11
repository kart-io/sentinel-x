// Package json provides a high-performance JSON serialization wrapper.
// It automatically uses sonic for supported architectures (amd64/arm64) and
// falls back to standard encoding/json for other platforms.
package json

import (
	stdjson "encoding/json"
	"io"
	"runtime"
	"sync/atomic"

	"github.com/bytedance/sonic"
)

// jsonAPI 封装 JSON 序列化/反序列化函数，用于原子切换
type jsonAPI struct {
	marshal    func(v interface{}) ([]byte, error)
	unmarshal  func(data []byte, v interface{}) error
	newEncoder func(w io.Writer) Encoder
	newDecoder func(r io.Reader) Decoder
}

var (
	// currentAPI 存储当前使用的 JSON API，使用原子操作保证并发安全
	currentAPI atomic.Value // stores *jsonAPI

	// usingSonic 标识是否使用 sonic（初始化后只读，无需原子保护）
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
	var api *jsonAPI

	// Sonic only supports amd64 and arm64 architectures
	if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
		// Use sonic's default configuration (balances performance and compatibility)
		api = &jsonAPI{
			marshal:   sonic.Marshal,
			unmarshal: sonic.Unmarshal,
			newEncoder: func(w io.Writer) Encoder {
				return sonic.ConfigDefault.NewEncoder(w)
			},
			newDecoder: func(r io.Reader) Decoder {
				return sonic.ConfigDefault.NewDecoder(r)
			},
		}
		usingSonic = true
	} else {
		// Fallback to standard library for unsupported architectures
		api = &jsonAPI{
			marshal:   stdjson.Marshal,
			unmarshal: stdjson.Unmarshal,
			newEncoder: func(w io.Writer) Encoder {
				return stdjson.NewEncoder(w)
			},
			newDecoder: func(r io.Reader) Decoder {
				return stdjson.NewDecoder(r)
			},
		}
		usingSonic = false
	}

	currentAPI.Store(api)
}

// getAPI 返回当前 JSON API（线程安全）
func getAPI() *jsonAPI {
	return currentAPI.Load().(*jsonAPI)
}

// Marshal encodes v into JSON bytes.
// Uses sonic on amd64/arm64, otherwise falls back to encoding/json.
func Marshal(v interface{}) ([]byte, error) {
	return getAPI().marshal(v)
}

// Unmarshal decodes JSON bytes into v.
// Uses sonic on amd64/arm64, otherwise falls back to encoding/json.
func Unmarshal(data []byte, v interface{}) error {
	return getAPI().unmarshal(data, v)
}

// NewEncoder creates a new JSON encoder for the writer.
func NewEncoder(w io.Writer) Encoder {
	return getAPI().newEncoder(w)
}

// NewDecoder creates a new JSON decoder for the reader.
func NewDecoder(r io.Reader) Decoder {
	return getAPI().newDecoder(r)
}

// ConfigFastestMode switches to sonic's fastest mode.
// This mode disables some safety checks for maximum performance.
// Only affects sonic implementation; no-op for standard library fallback.
//
// Use this when:
//   - You control the input data and trust it
//   - Maximum throughput is critical
//   - The data structures are simple and well-tested
//
// DO NOT use this when:
//   - Handling untrusted external input
//   - Data structures have complex validation requirements
func ConfigFastestMode() {
	if usingSonic {
		api := sonic.ConfigFastest
		newAPI := &jsonAPI{
			marshal:   api.Marshal,
			unmarshal: api.Unmarshal,
			newEncoder: func(w io.Writer) Encoder {
				return api.NewEncoder(w)
			},
			newDecoder: func(r io.Reader) Decoder {
				return api.NewDecoder(r)
			},
		}
		currentAPI.Store(newAPI) // 原子操作，线程安全
	}
}

// ConfigStandardMode switches to sonic's standard mode.
// This is the default mode with balanced performance and safety.
// Only affects sonic implementation; no-op for standard library fallback.
func ConfigStandardMode() {
	if usingSonic {
		api := sonic.ConfigDefault
		newAPI := &jsonAPI{
			marshal:   api.Marshal,
			unmarshal: api.Unmarshal,
			newEncoder: func(w io.Writer) Encoder {
				return api.NewEncoder(w)
			},
			newDecoder: func(r io.Reader) Decoder {
				return api.NewDecoder(r)
			},
		}
		currentAPI.Store(newAPI) // 原子操作，线程安全
	}
}

// IsUsingSonic returns true if sonic is being used for JSON operations.
func IsUsingSonic() bool {
	return usingSonic
}
