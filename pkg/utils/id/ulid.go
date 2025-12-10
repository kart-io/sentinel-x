package id

import (
	"crypto/rand"
	"io"
	"sync"
	"time"
)

const (
	// ULID encoding alphabet (Crockford's Base32)
	ulidAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

	// ULID lengths
	ulidEncodedLen = 26
	ulidTimeLen    = 10
	ulidRandomLen  = 16
)

// ULIDGenerator generates Universally Unique Lexicographically Sortable Identifiers.
// ULID format: ttttttttttrrrrrrrrrrrrrrr (26 characters)
//   - t: timestamp (48 bits, milliseconds since Unix epoch)
//   - r: randomness (80 bits)
type ULIDGenerator struct {
	mu       sync.Mutex
	reader   io.Reader
	lastTime int64
	lastRand [10]byte
}

// ULIDOption is a functional option for ULIDGenerator.
type ULIDOption func(*ULIDGenerator)

// WithULIDReader sets a custom random reader for ULID generation.
func WithULIDReader(r io.Reader) ULIDOption {
	return func(g *ULIDGenerator) {
		g.reader = r
	}
}

// NewULIDGenerator creates a new ULID generator.
func NewULIDGenerator(opts ...ULIDOption) *ULIDGenerator {
	g := &ULIDGenerator{
		reader: rand.Reader,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// Generate creates a new ULID string.
func (g *ULIDGenerator) Generate() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli()

	// Handle monotonicity
	if now == g.lastTime {
		// Increment random part
		g.incrementRandom()
	} else {
		// New timestamp, generate new random
		g.lastTime = now
		_, _ = io.ReadFull(g.reader, g.lastRand[:])
	}

	return g.encode(now, g.lastRand)
}

// GenerateN creates n ULID strings.
func (g *ULIDGenerator) GenerateN(n int) []string {
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ids[i] = g.Generate()
	}
	return ids
}

// incrementRandom increments the random part to maintain monotonicity.
func (g *ULIDGenerator) incrementRandom() {
	for i := len(g.lastRand) - 1; i >= 0; i-- {
		g.lastRand[i]++
		if g.lastRand[i] != 0 {
			return
		}
	}
	// Overflow, generate new random
	_, _ = io.ReadFull(g.reader, g.lastRand[:])
}

// encode encodes timestamp and random bytes into ULID string.
func (g *ULIDGenerator) encode(timestamp int64, random [10]byte) string {
	var ulid [ulidEncodedLen]byte

	// Encode timestamp (6 bytes -> 10 chars)
	ulid[0] = ulidAlphabet[(timestamp>>45)&0x1f]
	ulid[1] = ulidAlphabet[(timestamp>>40)&0x1f]
	ulid[2] = ulidAlphabet[(timestamp>>35)&0x1f]
	ulid[3] = ulidAlphabet[(timestamp>>30)&0x1f]
	ulid[4] = ulidAlphabet[(timestamp>>25)&0x1f]
	ulid[5] = ulidAlphabet[(timestamp>>20)&0x1f]
	ulid[6] = ulidAlphabet[(timestamp>>15)&0x1f]
	ulid[7] = ulidAlphabet[(timestamp>>10)&0x1f]
	ulid[8] = ulidAlphabet[(timestamp>>5)&0x1f]
	ulid[9] = ulidAlphabet[timestamp&0x1f]

	// Encode random (10 bytes -> 16 chars)
	ulid[10] = ulidAlphabet[(random[0]>>3)&0x1f]
	ulid[11] = ulidAlphabet[((random[0]<<2)|(random[1]>>6))&0x1f]
	ulid[12] = ulidAlphabet[(random[1]>>1)&0x1f]
	ulid[13] = ulidAlphabet[((random[1]<<4)|(random[2]>>4))&0x1f]
	ulid[14] = ulidAlphabet[((random[2]<<1)|(random[3]>>7))&0x1f]
	ulid[15] = ulidAlphabet[(random[3]>>2)&0x1f]
	ulid[16] = ulidAlphabet[((random[3]<<3)|(random[4]>>5))&0x1f]
	ulid[17] = ulidAlphabet[random[4]&0x1f]
	ulid[18] = ulidAlphabet[(random[5]>>3)&0x1f]
	ulid[19] = ulidAlphabet[((random[5]<<2)|(random[6]>>6))&0x1f]
	ulid[20] = ulidAlphabet[(random[6]>>1)&0x1f]
	ulid[21] = ulidAlphabet[((random[6]<<4)|(random[7]>>4))&0x1f]
	ulid[22] = ulidAlphabet[((random[7]<<1)|(random[8]>>7))&0x1f]
	ulid[23] = ulidAlphabet[(random[8]>>2)&0x1f]
	ulid[24] = ulidAlphabet[((random[8]<<3)|(random[9]>>5))&0x1f]
	ulid[25] = ulidAlphabet[random[9]&0x1f]

	return string(ulid[:])
}

// ULID represents a parsed ULID.
type ULID struct {
	str       string
	timestamp int64
	random    [10]byte
}

// ParseULID parses a ULID string.
func ParseULID(s string) (ULID, error) {
	if len(s) != ulidEncodedLen {
		return ULID{}, ErrInvalidULID
	}

	// Decode timestamp
	var timestamp int64
	for i := 0; i < ulidTimeLen; i++ {
		v := decodeChar(s[i])
		if v == 0xff {
			return ULID{}, ErrInvalidULID
		}
		timestamp = (timestamp << 5) | int64(v)
	}

	// Decode random
	var random [10]byte
	idx := ulidTimeLen
	random[0] = (decodeChar(s[idx])<<3 | decodeChar(s[idx+1])>>2)
	random[1] = (decodeChar(s[idx+1])<<6 | decodeChar(s[idx+2])<<1 | decodeChar(s[idx+3])>>4)
	random[2] = (decodeChar(s[idx+3])<<4 | decodeChar(s[idx+4])>>1)
	random[3] = (decodeChar(s[idx+4])<<7 | decodeChar(s[idx+5])<<2 | decodeChar(s[idx+6])>>3)
	random[4] = (decodeChar(s[idx+6])<<5 | decodeChar(s[idx+7]))
	random[5] = (decodeChar(s[idx+8])<<3 | decodeChar(s[idx+9])>>2)
	random[6] = (decodeChar(s[idx+9])<<6 | decodeChar(s[idx+10])<<1 | decodeChar(s[idx+11])>>4)
	random[7] = (decodeChar(s[idx+11])<<4 | decodeChar(s[idx+12])>>1)
	random[8] = (decodeChar(s[idx+12])<<7 | decodeChar(s[idx+13])<<2 | decodeChar(s[idx+14])>>3)
	random[9] = (decodeChar(s[idx+14])<<5 | decodeChar(s[idx+15]))

	return ULID{
		str:       s,
		timestamp: timestamp,
		random:    random,
	}, nil
}

// decodeChar decodes a single character.
func decodeChar(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'A' && c <= 'H':
		return c - 'A' + 10
	case c >= 'J' && c <= 'K':
		return c - 'J' + 18
	case c >= 'M' && c <= 'N':
		return c - 'M' + 20
	case c >= 'P' && c <= 'T':
		return c - 'P' + 22
	case c >= 'V' && c <= 'Z':
		return c - 'V' + 27
	case c >= 'a' && c <= 'h':
		return c - 'a' + 10
	case c >= 'j' && c <= 'k':
		return c - 'j' + 18
	case c >= 'm' && c <= 'n':
		return c - 'm' + 20
	case c >= 'p' && c <= 't':
		return c - 'p' + 22
	case c >= 'v' && c <= 'z':
		return c - 'v' + 27
	default:
		return 0xff
	}
}

// String returns the ULID string.
func (u ULID) String() string {
	return u.str
}

// Time returns the time when this ULID was generated.
func (u ULID) Time() time.Time {
	return time.UnixMilli(u.timestamp)
}

// Timestamp returns the Unix timestamp in milliseconds.
func (u ULID) Timestamp() int64 {
	return u.timestamp
}

// IsValidULID checks if a string is a valid ULID format.
func IsValidULID(s string) bool {
	_, err := ParseULID(s)
	return err == nil
}
