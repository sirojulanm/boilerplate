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

// HTTP routes and headers.
const (
	RequestIDHeader = "X-Request-ID"
	RequestIDKey    = "request_id"

	HealthPath    = "/api/v1/health"
	SwaggerPath   = "/swagger/*any"
	SwaggerUIPath = "/swagger/index.html"
)

// HTTP error messages.
const (
	ErrRouteNotFound    = "route not found"
	ErrMethodNotAllowed = "method not allowed"
	ErrInternalServer   = "internal server error"
)
