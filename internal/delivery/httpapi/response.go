package httpapi

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error string `json:"error" example:"internal server error"`
}
