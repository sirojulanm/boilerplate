package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirojulanam/boilerplate/internal/domain"
)

// ErrorBody carries a stable machine code plus a human-readable message.
type ErrorBody struct {
	Code    string `json:"code" example:"invalid_credentials"`
	Message string `json:"message" example:"invalid email or password"`
}

// ErrorResponse is the error envelope shared by every endpoint.
type ErrorResponse struct {
	Success   bool      `json:"success" example:"false"`
	Error     ErrorBody `json:"error"`
	RequestID string    `json:"request_id" example:"7f3a9c1d2e4b5a6f"`
}

// HealthResponse is the success envelope for the health endpoint.
type HealthResponse struct {
	Success   bool          `json:"success" example:"true"`
	Data      domain.Health `json:"data"`
	RequestID string        `json:"request_id" example:"7f3a9c1d2e4b5a6f"`
}

// UserResponse is the success envelope for register and profile responses.
type UserResponse struct {
	Success   bool        `json:"success" example:"true"`
	Data      domain.User `json:"data"`
	RequestID string      `json:"request_id" example:"7f3a9c1d2e4b5a6f"`
}

// TokenData is the payload returned by the token endpoint.
type TokenData struct {
	AccessToken string      `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	TokenType   string      `json:"token_type" example:"Bearer"`
	ExpiresIn   int64       `json:"expires_in" example:"3600"`
	ExpiresAt   time.Time   `json:"expires_at" example:"2026-10-27T23:00:00Z"`
	User        domain.User `json:"user"`
}

// TokenResponse is the success envelope for the token endpoint.
type TokenResponse struct {
	Success   bool      `json:"success" example:"true"`
	Data      TokenData `json:"data"`
	RequestID string    `json:"request_id" example:"7f3a9c1d2e4b5a6f"`
}

func requestID(c *gin.Context) string {
	return c.GetString(constant.RequestIDKey)
}

func respondHealth(c *gin.Context, status int, health domain.Health) {
	c.JSON(status, HealthResponse{Success: true, Data: health, RequestID: requestID(c)})
}

func respondUser(c *gin.Context, status int, user domain.User) {
	c.JSON(status, UserResponse{Success: true, Data: user, RequestID: requestID(c)})
}

func respondToken(c *gin.Context, status int, data TokenData) {
	c.JSON(status, TokenResponse{Success: true, Data: data, RequestID: requestID(c)})
}

func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Success:   false,
		Error:     ErrorBody{Code: code, Message: message},
		RequestID: requestID(c),
	})
}

// respondBindingError maps request decoding failures to the shared error
// envelope. Oversized bodies are reported as 413 instead of a generic 400.
func respondBindingError(c *gin.Context, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		respondError(c, http.StatusRequestEntityTooLarge, constant.CodeRequestTooLarge, constant.ErrRequestTooLarge)
		return
	}
	respondError(c, http.StatusBadRequest, constant.CodeInvalidRequest, constant.ErrInvalidRequest)
}
