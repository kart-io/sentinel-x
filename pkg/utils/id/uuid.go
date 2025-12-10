package id

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

// UUIDGenerator generates UUID v4 identifiers.
type UUIDGenerator struct {
	reader io.Reader
}

// UUIDOption is a functional option for UUIDGenerator.
type UUIDOption func(*UUIDGenerator)

// WithReader sets a custom random reader for UUID generation.
func WithReader(r io.Reader) UUIDOption {
	return func(g *UUIDGenerator) {
		g.reader = r
	}
}

// NewUUIDGenerator creates a new UUID v4 generator.
func NewUUIDGenerator(opts ...UUIDOption) *UUIDGenerator {
	g := &UUIDGenerator{
		reader: rand.Reader,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// Generate creates a new UUID v4 string.
// Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
// where x is any hexadecimal digit and y is one of 8, 9, A, or B.
// Panics if the random source fails (should never happen with crypto/rand).
func (g *UUIDGenerator) Generate() string {
	uuid, err := g.GenerateE()
	if err != nil {
		panic("id: failed to generate UUID: " + err.Error())
	}
	return uuid
}

// GenerateE creates a new UUID v4 string, returning an error on failure.
// Use this variant when you need explicit error handling.
func (g *UUIDGenerator) GenerateE() (string, error) {
	var uuid [16]byte

	_, err := io.ReadFull(g.reader, uuid[:])
	if err != nil {
		return "", err
	}

	// Set version 4 (random)
	uuid[6] = (uuid[6] & 0x0f) | 0x40

	// Set variant (RFC 4122)
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return formatUUID(uuid), nil
}

// GenerateN creates n UUID v4 strings.
func (g *UUIDGenerator) GenerateN(n int) []string {
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ids[i] = g.Generate()
	}
	return ids
}

// formatUUID formats a 16-byte array as a UUID string.
func formatUUID(uuid [16]byte) string {
	buf := make([]byte, 36)

	hex.Encode(buf[0:8], uuid[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], uuid[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], uuid[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], uuid[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], uuid[10:16])

	return string(buf)
}

// ParseUUID parses a UUID string and returns the bytes.
func ParseUUID(s string) ([16]byte, error) {
	var uuid [16]byte

	if len(s) != 36 {
		return uuid, ErrInvalidUUID
	}

	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return uuid, ErrInvalidUUID
	}

	// Remove dashes and decode
	hexStr := s[0:8] + s[9:13] + s[14:18] + s[19:23] + s[24:36]
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return uuid, ErrInvalidUUID
	}

	copy(uuid[:], b)
	return uuid, nil
}

// IsValidUUID checks if a string is a valid UUID format.
func IsValidUUID(s string) bool {
	_, err := ParseUUID(s)
	return err == nil
}
