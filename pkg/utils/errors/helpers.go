package errors

// FromError converts any error to Errno.
// If err is already an Errno, returns it directly.
// Otherwise, wraps it as ErrInternal.
func FromError(err error) *Errno {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Errno); ok {
		return e
	}
	return ErrInternal.WithCause(err)
}

// IsCode checks if the error has the given error code.
func IsCode(err error, code int) bool {
	if e, ok := err.(*Errno); ok {
		return e.Code == code
	}
	return false
}

// GetCode returns the error code from an error.
// Returns -1 if the error is not an Errno.
func GetCode(err error) int {
	if e, ok := err.(*Errno); ok {
		return e.Code
	}
	return -1
}
