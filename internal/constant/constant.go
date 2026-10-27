package constant

// Application defaults. Override them with environment variables or build flags
// when the boilerplate is reused for another service.
const (
	DefaultServiceName = "boilerplate"
	DefaultVersion     = "1.0.0"
	DefaultPort        = "1111"
)

// Logging defaults.
const (
	DefaultLogLevel    = "info"
	DefaultLogDir      = "logs"
	DefaultLogFilename = "app.log"
)

// Database defaults. These are development-only values; always provide real
// credentials through the environment in shared or production environments.
const (
	DefaultDBHost           = "localhost"
	DefaultDBPort           = "5432"
	DefaultDBUser           = "postgres"
	DefaultDBPassword       = "postgres"
	DefaultDBName           = "boilerplate"
	DefaultDBSSLMode        = "disable"
	DefaultDBMaxConns       = "10"
	DefaultDBConnectTimeout = "5s"
	DefaultDBAutoMigrate    = "false"
)

// HTTP routes, headers, and request limits.
const (
	RequestIDHeader = "X-Request-ID"
	RequestIDKey    = "request_id"

	HealthPath   = "/api/v1/health"
	RegisterPath = "/api/v1/auth/register"
	TokenPath    = "/api/v1/auth/token"
	MePath       = "/api/v1/auth/me"

	SwaggerPath   = "/swagger/*any"
	SwaggerUIPath = "/swagger/index.html"

	MaxRequestBodyBytes = 1 << 20
	BearerScheme        = "Bearer"
	AuthRealm           = "api"
	AuthUserKey         = "auth_user"
)

// Authentication defaults.
const (
	DefaultJWTIssuer   = "boilerplate"
	DefaultJWTAudience = "boilerplate-api"
	DefaultJWTTTL      = "1h"
	MinJWTSecretLength = 32
	BcryptCost         = 12
	MinPasswordLength  = 8
	MaxPasswordLength  = 72
	MaxEmailLength     = 254
	MaxNameLength      = 100
)

// Log event values make request, process, and response entries easy to filter.
const (
	LogEventRequest  = "request"
	LogEventProcess  = "process"
	LogEventResponse = "response"
)

// HTTP error messages.
const (
	ErrRouteNotFound      = "route not found"
	ErrMethodNotAllowed   = "method not allowed"
	ErrInternalServer     = "internal server error"
	ErrInvalidRequest     = "invalid request body"
	ErrRequestTooLarge    = "request body too large"
	ErrInvalidEmail       = "invalid email"
	ErrInvalidName        = "invalid name"
	ErrWeakPassword       = "password must be 8-72 characters"
	ErrEmailTaken         = "email already registered"
	ErrInvalidCredentials = "invalid email or password"
	ErrUnauthorized       = "unauthorized"
)

// Stable machine-readable error codes for the response envelope.
const (
	CodeRouteNotFound      = "route_not_found"
	CodeMethodNotAllowed   = "method_not_allowed"
	CodeInternalServer     = "internal_server_error"
	CodeInvalidRequest     = "invalid_request"
	CodeRequestTooLarge    = "request_too_large"
	CodeInvalidEmail       = "invalid_email"
	CodeInvalidName        = "invalid_name"
	CodeWeakPassword       = "weak_password"
	CodeEmailTaken         = "email_taken"
	CodeInvalidCredentials = "invalid_credentials"
	CodeUnauthorized       = "unauthorized"
)
