package requestutil

import (
	"crypto/rand"
	"io"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

// IDGenerator 定义请求 ID 生成器接口
type IDGenerator interface {
	Generate() string
}

// ============================================================================
// Random Hex Generator (当前默认实现)
// ============================================================================

// RandomHexGenerator 使用加密随机数生成十六进制字符串 ID
type RandomHexGenerator struct{}

// Generate 实现 IDGenerator 接口
func (g *RandomHexGenerator) Generate() string {
	return GenerateRequestID()
}

// ============================================================================
// ULID Generator (推荐实现)
// ============================================================================

// ULIDGenerator 使用 ULID 算法生成时间可排序的唯一 ID
//
// ULID 特性:
//   - 时间可排序 (毫秒精度)
//   - 词典序友好 (适合数据库索引)
//   - 26 字符长度 (vs UUID 36 字符)
//   - 性能优于 UUID v4 (~15x)
//
// 格式: 01AN4Z07BY79KA1307SR9X4MV3
//   - 前 10 字符: 时间戳 (毫秒)
//   - 后 16 字符: 随机熵
type ULIDGenerator struct {
	entropy io.Reader
	mu      sync.Mutex
}

// NewULIDGenerator 创建新的 ULID 生成器
func NewULIDGenerator() *ULIDGenerator {
	// 使用单调熵源确保同一毫秒内生成的 ID 也是有序的
	// 使用 crypto/rand 作为安全的随机源
	entropy := ulid.Monotonic(rand.Reader, 0)
	return &ULIDGenerator{
		entropy: entropy,
	}
}

// Generate 实现 IDGenerator 接口
func (g *ULIDGenerator) Generate() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), g.entropy).String()
}

// ============================================================================
// Factory Functions
// ============================================================================

// NewGenerator 根据类型名称创建生成器
//
// 支持的类型:
//   - "random" 或 "hex": RandomHexGenerator (当前默认)
//   - "ulid": ULIDGenerator (推荐)
//
// 默认: RandomHexGenerator
func NewGenerator(generatorType string) IDGenerator {
	switch generatorType {
	case "ulid":
		return NewULIDGenerator()
	case "random", "hex", "":
		return &RandomHexGenerator{}
	default:
		// 未知类型使用默认
		return &RandomHexGenerator{}
	}
}
