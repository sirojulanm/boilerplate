package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirojulanam/boilerplate/internal/domain"
	"github.com/sirojulanam/boilerplate/internal/usecase"
	"github.com/sirupsen/logrus"
)

// RegisterRequest is the public registration payload.
type RegisterRequest struct {
	Name     string `json:"name" example:"Jane Doe"`
	Email    string `json:"email" example:"jane@example.com"`
	Password string `json:"password" example:"correct-horse-battery"`
}

// TokenRequest is the credential payload for token generation.
type TokenRequest struct {
	Email    string `json:"email" example:"jane@example.com"`
	Password string `json:"password" example:"correct-horse-battery"`
}

// AuthHandler exposes registration, token generation, and the protected
// profile endpoint.
type AuthHandler struct {
	auth usecase.Auth
	log  *logrus.Logger
}

// NewAuthHandler wires the authentication usecase into HTTP.
func NewAuthHandler(auth usecase.Auth, log *logrus.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, log: log}
}

// Register godoc
// @Summary      Register a user
// @Description  Creates an account with a bcrypt-hashed password.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterRequest  true  "Registration payload"
// @Success      201      {object}  UserResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      409      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var request RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.log.WithFields(logrus.Fields{
			"event":      constant.LogEventProcess,
			"request_id": requestID(c),
			"action":     "register",
			"reason":     "invalid json body",
		}).Warn("register rejected")
		respondBindingError(c, err)
		return
	}

	h.log.WithFields(logrus.Fields{
		"event":      constant.LogEventProcess,
		"request_id": requestID(c),
		"action":     "register",
	}).Info("register started")

	user, err := h.auth.Register(c.Request.Context(), usecase.RegisterInput{
		Name:     request.Name,
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		h.writeError(c, "register", err)
		return
	}

	h.log.WithFields(logrus.Fields{
		"event":      constant.LogEventProcess,
		"request_id": requestID(c),
		"action":     "register",
		"user_id":    user.ID,
	}).Info("register completed")

	respondUser(c, http.StatusCreated, user)
}

// Token godoc
// @Summary      Generate an access token
// @Description  Verifies email and password, then returns a signed JWT.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      TokenRequest  true  "Credential payload"
// @Success      200      {object}  TokenResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /auth/token [post]
func (h *AuthHandler) Token(c *gin.Context) {
	var request TokenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.log.WithFields(logrus.Fields{
			"event":      constant.LogEventProcess,
			"request_id": requestID(c),
			"action":     "token",
			"reason":     "invalid json body",
		}).Warn("token request rejected")
		respondBindingError(c, err)
		return
	}

	h.log.WithFields(logrus.Fields{
		"event":      constant.LogEventProcess,
		"request_id": requestID(c),
		"action":     "token",
	}).Info("token generation started")

	result, err := h.auth.GenerateToken(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		h.writeError(c, "token", err)
		return
	}

	h.log.WithFields(logrus.Fields{
		"event":      constant.LogEventProcess,
		"request_id": requestID(c),
		"action":     "token",
		"user_id":    result.User.ID,
		"expires_at": result.Token.ExpiresAt,
	}).Info("token generation completed")

	respondToken(c, http.StatusOK, TokenData{
		AccessToken: result.Token.AccessToken,
		TokenType:   result.Token.TokenType,
		ExpiresIn:   result.Token.ExpiresIn,
		ExpiresAt:   result.Token.ExpiresAt,
		User:        result.User,
	})
}

// Me godoc
// @Summary      Current user
// @Description  Returns the account identified by the Bearer token.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  UserResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	identity, ok := currentIdentity(c)
	if !ok {
		abortUnauthorized(c)
		return
	}

	h.log.WithFields(logrus.Fields{
		"event":      constant.LogEventProcess,
		"request_id": requestID(c),
		"action":     "profile",
		"user_id":    identity.UserID,
	}).Info("profile lookup started")

	user, err := h.auth.Profile(c.Request.Context(), identity.UserID)
	if err != nil {
		h.writeError(c, "profile", err)
		return
	}

	h.log.WithFields(logrus.Fields{
		"event":      constant.LogEventProcess,
		"request_id": requestID(c),
		"action":     "profile",
		"user_id":    user.ID,
	}).Info("profile lookup completed")

	respondUser(c, http.StatusOK, user)
}

func currentIdentity(c *gin.Context) (domain.Identity, bool) {
	value, exists := c.Get(constant.AuthUserKey)
	if !exists {
		return domain.Identity{}, false
	}
	identity, ok := value.(domain.Identity)
	return identity, ok
}

func (h *AuthHandler) writeError(c *gin.Context, action string, err error) {
	status, code, message := authErrorResponse(err)

	fields := logrus.Fields{
		"event":      constant.LogEventProcess,
		"request_id": requestID(c),
		"action":     action,
		"error":      err.Error(),
	}
	if status >= http.StatusInternalServerError {
		h.log.WithFields(fields).Error("request failed")
	} else {
		h.log.WithFields(fields).Warn("request failed")
	}

	respondError(c, status, code, message)
}

func authErrorResponse(err error) (int, string, string) {
	switch {
	case errors.Is(err, domain.ErrEmailTaken):
		return http.StatusConflict, constant.CodeEmailTaken, constant.ErrEmailTaken
	case errors.Is(err, domain.ErrInvalidEmail):
		return http.StatusBadRequest, constant.CodeInvalidEmail, constant.ErrInvalidEmail
	case errors.Is(err, domain.ErrInvalidName):
		return http.StatusBadRequest, constant.CodeInvalidName, constant.ErrInvalidName
	case errors.Is(err, domain.ErrWeakPassword):
		return http.StatusBadRequest, constant.CodeWeakPassword, constant.ErrWeakPassword
	case errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrInvalidToken):
		return http.StatusUnauthorized, constant.CodeInvalidCredentials, constant.ErrInvalidCredentials
	default:
		return http.StatusInternalServerError, constant.CodeInternalServer, constant.ErrInternalServer
	}
}
