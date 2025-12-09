package id

import (
	"strconv"
	"sync"
	"time"
)

const (
	// Snowflake bit allocation
	// 1 bit sign | 41 bits timestamp | 10 bits node | 12 bits sequence

	snowflakeEpoch     = int64(1704067200000) // 2024-01-01 00:00:00 UTC in milliseconds
	snowflakeNodeBits  = 10
	snowflakeSeqBits   = 12
	snowflakeMaxNode   = (1 << snowflakeNodeBits) - 1
	snowflakeMaxSeq    = (1 << snowflakeSeqBits) - 1
	snowflakeTimeShift = snowflakeNodeBits + snowflakeSeqBits
	snowflakeNodeShift = snowflakeSeqBits
)

// SnowflakeGenerator generates Twitter Snowflake IDs.
// The ID is a 64-bit integer composed of:
//   - 1 bit: sign (always 0)
//   - 41 bits: timestamp in milliseconds (69 years from epoch)
//   - 10 bits: node ID (0-1023)
//   - 12 bits: sequence number (0-4095)
type SnowflakeGenerator struct {
	mu       sync.Mutex
	epoch    int64
	nodeID   int64
	lastTime int64
	sequence int64
	timeFunc func() int64
}

// SnowflakeOption is a functional option for SnowflakeGenerator.
type SnowflakeOption func(*SnowflakeGenerator)

// WithNodeID sets the node ID (0-1023).
func WithNodeID(nodeID int64) SnowflakeOption {
	return func(g *SnowflakeGenerator) {
		g.nodeID = nodeID
	}
}

// WithEpoch sets a custom epoch timestamp in milliseconds.
func WithEpoch(epoch int64) SnowflakeOption {
	return func(g *SnowflakeGenerator) {
		g.epoch = epoch
	}
}

// WithTimeFunc sets a custom time function (for testing).
func WithTimeFunc(f func() int64) SnowflakeOption {
	return func(g *SnowflakeGenerator) {
		g.timeFunc = f
	}
}

// NewSnowflakeGenerator creates a new Snowflake ID generator.
func NewSnowflakeGenerator(opts ...SnowflakeOption) (*SnowflakeGenerator, error) {
	g := &SnowflakeGenerator{
		epoch:    snowflakeEpoch,
		nodeID:   0,
		lastTime: 0,
		sequence: 0,
		timeFunc: func() int64 {
			return time.Now().UnixMilli()
		},
	}

	for _, opt := range opts {
		opt(g)
	}

	if g.nodeID < 0 || g.nodeID > snowflakeMaxNode {
		return nil, ErrInvalidNodeID
	}

	return g, nil
}

// Generate creates a new Snowflake ID string.
func (g *SnowflakeGenerator) Generate() string {
	id := g.GenerateInt64()
	return strconv.FormatInt(id, 10)
}

// GenerateN creates n Snowflake ID strings.
func (g *SnowflakeGenerator) GenerateN(n int) []string {
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ids[i] = g.Generate()
	}
	return ids
}

// GenerateInt64 creates a new Snowflake ID as int64.
func (g *SnowflakeGenerator) GenerateInt64() int64 {
	g.mu.Lock()

	now := g.timeFunc()

	// Handle clock moving backward - release lock while waiting
	for now < g.lastTime {
		g.mu.Unlock()
		time.Sleep(time.Millisecond)
		g.mu.Lock()
		now = g.timeFunc()
	}

	if now == g.lastTime {
		g.sequence = (g.sequence + 1) & snowflakeMaxSeq
		if g.sequence == 0 {
			// Sequence overflow, wait for next millisecond - release lock while waiting
			for now <= g.lastTime {
				g.mu.Unlock()
				time.Sleep(time.Millisecond)
				g.mu.Lock()
				now = g.timeFunc()
			}
		}
	} else {
		g.sequence = 0
	}

	g.lastTime = now

	// Compose the ID
	id := ((now - g.epoch) << snowflakeTimeShift) |
		(g.nodeID << snowflakeNodeShift) |
		g.sequence

	g.mu.Unlock()
	return id
}

// ParseSnowflake extracts components from a Snowflake ID.
func ParseSnowflake(id int64) SnowflakeID {
	return SnowflakeID{
		ID:        id,
		Timestamp: (id >> snowflakeTimeShift) + snowflakeEpoch,
		NodeID:    (id >> snowflakeNodeShift) & snowflakeMaxNode,
		Sequence:  id & snowflakeMaxSeq,
	}
}

// ParseSnowflakeString parses a Snowflake ID string.
func ParseSnowflakeString(s string) (SnowflakeID, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return SnowflakeID{}, err
	}
	return ParseSnowflake(id), nil
}

// SnowflakeID represents a parsed Snowflake ID.
type SnowflakeID struct {
	ID        int64 // Original ID
	Timestamp int64 // Unix timestamp in milliseconds
	NodeID    int64 // Node/machine ID
	Sequence  int64 // Sequence number
}

// Time returns the time when this ID was generated.
func (s SnowflakeID) Time() time.Time {
	return time.UnixMilli(s.Timestamp)
}

// String returns the string representation of the ID.
func (s SnowflakeID) String() string {
	return strconv.FormatInt(s.ID, 10)
}
