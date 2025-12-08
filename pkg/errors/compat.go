package errors

// ============================================================================
// Legacy Error Code Compatibility Layer
// ============================================================================
//
// This file provides backward compatibility for the legacy error codes (1000-5999)
// used in the previous version. These mappings allow gradual migration to the
// new error code system (AABBCCC format).
//
// DEPRECATION NOTICE: Legacy error codes are deprecated and will be removed
// in a future version. Please migrate to the new error code system.
//
// Migration Timeline:
// - v1.x: Both systems supported (current)
// - v2.0: Legacy codes deprecated, warnings logged
// - v3.0: Legacy codes removed
//
// ============================================================================

// Legacy error codes (deprecated, use new error codes instead)
const (
	// LegacyCodeSuccess is deprecated, use 0 instead.
	// Deprecated: Use errors.OK.Code instead.
	LegacyCodeSuccess = 0

	// Client errors (1000-1999)
	// Deprecated: Use new error codes (0001xxx) instead.
	LegacyCodeBadRequest       = 1000
	LegacyCodeInvalidParam     = 1001
	LegacyCodeMissingParam     = 1002
	LegacyCodeInvalidFormat    = 1003
	LegacyCodeValidationFailed = 1004
	LegacyCodeTooManyRequests  = 1005

	// Authentication errors (2000-2999)
	// Deprecated: Use new error codes (0002xxx) instead.
	LegacyCodeUnauthorized      = 2000
	LegacyCodeInvalidToken      = 2001
	LegacyCodeTokenExpired      = 2002
	LegacyCodeInvalidCredential = 2003

	// Authorization errors (3000-3999)
	// Deprecated: Use new error codes (0003xxx) instead.
	LegacyCodeForbidden      = 3000
	LegacyCodeNoPermission   = 3001
	LegacyCodeResourceLocked = 3002

	// Not found errors (4000-4999)
	// Deprecated: Use new error codes (0004xxx) instead.
	LegacyCodeNotFound       = 4000
	LegacyCodeUserNotFound   = 4001
	LegacyCodeRecordNotFound = 4002

	// Server errors (5000-5999)
	// Deprecated: Use new error codes (0007xxx) instead.
	LegacyCodeInternalError   = 5000
	LegacyCodeDatabaseError   = 5001
	LegacyCodeCacheError      = 5002
	LegacyCodeExternalService = 5003
	LegacyCodeTimeout         = 5004
)

// legacyToNewCodeMap maps legacy error codes to new error codes.
var legacyToNewCodeMap = map[int]int{
	// Success
	LegacyCodeSuccess: OK.Code,

	// Client errors
	LegacyCodeBadRequest:       ErrBadRequest.Code,
	LegacyCodeInvalidParam:     ErrInvalidParam.Code,
	LegacyCodeMissingParam:     ErrMissingParam.Code,
	LegacyCodeInvalidFormat:    ErrInvalidFormat.Code,
	LegacyCodeValidationFailed: ErrValidationFailed.Code,
	LegacyCodeTooManyRequests:  ErrTooManyRequests.Code,

	// Authentication errors
	LegacyCodeUnauthorized:      ErrUnauthorized.Code,
	LegacyCodeInvalidToken:      ErrInvalidToken.Code,
	LegacyCodeTokenExpired:      ErrTokenExpired.Code,
	LegacyCodeInvalidCredential: ErrInvalidCredentials.Code,

	// Authorization errors
	LegacyCodeForbidden:      ErrForbidden.Code,
	LegacyCodeNoPermission:   ErrNoPermission.Code,
	LegacyCodeResourceLocked: ErrResourceLocked.Code,

	// Not found errors
	LegacyCodeNotFound:       ErrNotFound.Code,
	LegacyCodeUserNotFound:   ErrUserNotFound.Code,
	LegacyCodeRecordNotFound: ErrRecordNotFound.Code,

	// Server errors
	LegacyCodeInternalError:   ErrInternal.Code,
	LegacyCodeDatabaseError:   ErrDatabase.Code,
	LegacyCodeCacheError:      ErrCache.Code,
	LegacyCodeExternalService: ErrServiceUnavailable.Code,
	LegacyCodeTimeout:         ErrTimeout.Code,
}

// newToLegacyCodeMap maps new error codes to legacy error codes (for backward compatibility).
var newToLegacyCodeMap = map[int]int{}

func init() {
	// Build reverse mapping
	for legacy, new := range legacyToNewCodeMap {
		newToLegacyCodeMap[new] = legacy
	}
}

// LegacyToNewCode converts a legacy error code to the new error code.
// Returns the legacy code if no mapping exists.
func LegacyToNewCode(legacyCode int) int {
	if newCode, ok := legacyToNewCodeMap[legacyCode]; ok {
		return newCode
	}
	return legacyCode
}

// NewToLegacyCode converts a new error code to the legacy error code.
// Returns the new code if no mapping exists.
func NewToLegacyCode(newCode int) int {
	if legacyCode, ok := newToLegacyCodeMap[newCode]; ok {
		return legacyCode
	}
	return newCode
}

// IsLegacyCode checks if the code is a legacy error code.
func IsLegacyCode(code int) bool {
	_, ok := legacyToNewCodeMap[code]
	return ok
}

// FromLegacyCode returns the Errno for a legacy error code.
// Returns ErrUnknown if the code is not recognized.
func FromLegacyCode(code int) *Errno {
	newCode := LegacyToNewCode(code)
	if e, ok := Lookup(newCode); ok {
		return e
	}
	return ErrUnknown
}

// ============================================================================
// Deprecated Error Aliases (for backward compatibility)
// ============================================================================
// These aliases map legacy error variable names to new error variables.
// They are deprecated and will be removed in v3.0.

// Deprecated error aliases for backward compatibility.
// Use the new error variables instead.
var (
	// Deprecated: Use ErrInvalidCredentials instead.
	ErrInvalidCredential = ErrInvalidCredentials
)
