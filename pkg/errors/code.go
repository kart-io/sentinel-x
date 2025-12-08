// Package errors provides a unified error handling system for Sentinel-X.
//
// Error Code Format: AABBCCC (7 digits)
//
//   - AA:  Service/Module code (00-99)
//   - BB:  Category code (00-99)
//   - CCC: Sequence number (000-999)
//
// Service Codes (AA):
//
//   - 00: Common/Base errors (all services)
//   - 01: Gateway service
//   - 02: User service
//   - 03: Scheduler service
//   - 04: API service
//   - 05-09: Reserved for core services
//   - 10-19: Infrastructure errors
//   - 20-79: Business service errors
//   - 80-89: Internal service errors
//   - 90-99: Third-party service errors
//
// Category Codes (BB):
//
//   - 00: Success
//   - 01: Request/Validation errors (400)
//   - 02: Authentication errors (401)
//   - 03: Authorization errors (403)
//   - 04: Resource errors (404)
//   - 05: Conflict errors (409)
//   - 06: Rate limiting errors (429)
//   - 07: Internal errors (500)
//   - 08: Database errors (500)
//   - 09: Cache errors (500)
//   - 10: Network errors (502/503)
//   - 11: Timeout errors (504)
//   - 12: Configuration errors (500)
package errors

// Service codes (AA)
const (
	// ServiceCommon is for common/base errors shared by all services.
	ServiceCommon = 0

	// ServiceGateway is for gateway service.
	ServiceGateway = 1

	// ServiceUser is for user service.
	ServiceUser = 2

	// ServiceScheduler is for scheduler service.
	ServiceScheduler = 3

	// ServiceAPI is for API service.
	ServiceAPI = 4

	// ServiceInfraDB is for database infrastructure.
	ServiceInfraDB = 10

	// ServiceInfraCache is for cache infrastructure.
	ServiceInfraCache = 11

	// ServiceInfraMQ is for message queue infrastructure.
	ServiceInfraMQ = 12

	// ServiceThirdPartyPayment is for payment third-party service.
	ServiceThirdPartyPayment = 90

	// ServiceThirdPartySMS is for SMS third-party service.
	ServiceThirdPartySMS = 91

	// ServiceThirdPartyEmail is for email third-party service.
	ServiceThirdPartyEmail = 92

	// ServiceThirdPartyStorage is for storage third-party service.
	ServiceThirdPartyStorage = 93
)

// Category codes (BB)
const (
	// CategorySuccess indicates successful operation.
	CategorySuccess = 0

	// CategoryRequest indicates request/validation errors.
	CategoryRequest = 1

	// CategoryAuth indicates authentication errors.
	CategoryAuth = 2

	// CategoryPermission indicates authorization errors.
	CategoryPermission = 3

	// CategoryResource indicates resource not found errors.
	CategoryResource = 4

	// CategoryConflict indicates resource conflict errors.
	CategoryConflict = 5

	// CategoryRateLimit indicates rate limiting errors.
	CategoryRateLimit = 6

	// CategoryInternal indicates internal server errors.
	CategoryInternal = 7

	// CategoryDatabase indicates database errors.
	CategoryDatabase = 8

	// CategoryCache indicates cache errors.
	CategoryCache = 9

	// CategoryNetwork indicates network errors.
	CategoryNetwork = 10

	// CategoryTimeout indicates timeout errors.
	CategoryTimeout = 11

	// CategoryConfig indicates configuration errors.
	CategoryConfig = 12
)

// MakeCode creates an error code from service, category, and sequence.
// Format: AABBCCC where AA=service, BB=category, CCC=sequence
func MakeCode(service, category, sequence int) int {
	return service*100000 + category*1000 + sequence
}

// ParseCode parses an error code into service, category, and sequence.
func ParseCode(code int) (service, category, sequence int) {
	service = code / 100000
	category = (code % 100000) / 1000
	sequence = code % 1000
	return
}

// GetService returns the service code from an error code.
func GetService(code int) int {
	return code / 100000
}

// GetCategory returns the category code from an error code.
func GetCategory(code int) int {
	return (code % 100000) / 1000
}

// GetSequence returns the sequence number from an error code.
func GetSequence(code int) int {
	return code % 1000
}

// IsSuccess checks if the error code indicates success.
func IsSuccess(code int) bool {
	return code == 0
}

// IsClientError checks if the error code indicates a client error (4xx).
func IsClientError(code int) bool {
	category := GetCategory(code)
	return category >= CategoryRequest && category <= CategoryRateLimit
}

// IsServerError checks if the error code indicates a server error (5xx).
func IsServerError(code int) bool {
	category := GetCategory(code)
	return category >= CategoryInternal && category <= CategoryConfig
}
