package id

import "errors"

var (
	// ErrInvalidUUID is returned when a UUID string is invalid.
	ErrInvalidUUID = errors.New("invalid UUID format")

	// ErrInvalidULID is returned when a ULID string is invalid.
	ErrInvalidULID = errors.New("invalid ULID format")

	// ErrInvalidNodeID is returned when the node ID is out of range.
	ErrInvalidNodeID = errors.New("node ID must be between 0 and 1023")

	// ErrClockMovedBackward is returned when the system clock moves backward.
	ErrClockMovedBackward = errors.New("clock moved backward")

	// ErrSequenceOverflow is returned when the sequence number overflows.
	ErrSequenceOverflow = errors.New("sequence overflow")
)
