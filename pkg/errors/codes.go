package errors

const (
	// User Errors
	CodeUserNotFound         = "USER_NOT_FOUND"
	CodeUserUnauthorized     = "USER_UNAUTHORIZED"
	CodeUserAlreadyExists    = "USER_ALREADY_EXISTS"
	CodeInvalidCredentials   = "INVALID_CREDENTIALS" //nolint:gosec // not a credential
	CodeMissingRequiredField = "MISSING_REQUIRED_FIELD"
	CodeInvalidFieldFormat   = "INVALID_FIELD_FORMAT"
	CodeForbidden            = "FORBIDDEN"

	// Auth Errors
	CodeTokenExpired      = "TOKEN_EXPIRED"
	CodeTokenInvalid      = "TOKEN_INVALID"
	CodeInvalidPassword   = "INVALID_PASSWORD"
	CodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED" //nolint:gosec // not a credential

	// Database Errors
	CodeRecordNotFound          = "RECORD_NOT_FOUND"
	CodeDuplicateRecord         = "DUPLICATE_RECORD"
	CodeDatabaseConnectionError = "DATABASE_CONNECTION_ERROR"

	// Internal Errors
	CodeInternalServerError  = "INTERNAL_SERVER_ERROR"
	CodeNetworkError         = "NETWORK_ERROR"
	CodeInvalidConfiguration = "INVALID_CONFIGURATION"
	CodeUnexpectedError      = "UNEXPECTED_ERROR"
	CodeTimeoutError         = "TIMEOUT_ERROR"
)
