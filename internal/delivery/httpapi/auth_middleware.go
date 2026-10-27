package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirojulanam/boilerplate/internal/domain"
	"github.com/sirupsen/logrus"
)

// withAuthentication requires a valid Bearer token. It stores the resulting
// domain.Identity on the Gin context for protected handlers.
func withAuthentication(verifier domain.TokenVerifier, log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawToken, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			log.WithFields(logrus.Fields{
				"event":      constant.LogEventProcess,
				"request_id": c.GetString(constant.RequestIDKey),
				"path":       c.Request.URL.Path,
				"reason":     "missing bearer token",
			}).Warn("authentication failed")
			abortUnauthorized(c)
			return
		}

		identity, err := verifier.Verify(rawToken)
		if err != nil {
			log.WithFields(logrus.Fields{
				"event":      constant.LogEventProcess,
				"request_id": c.GetString(constant.RequestIDKey),
				"path":       c.Request.URL.Path,
				"reason":     "invalid bearer token",
			}).Warn("authentication failed")
			abortUnauthorized(c)
			return
		}

		c.Set(constant.AuthUserKey, identity)
		c.Next()
	}
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], constant.BearerScheme) {
		return "", false
	}
	return parts[1], true
}

func abortUnauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", fmt.Sprintf("%s realm=%q", constant.BearerScheme, constant.AuthRealm))
	c.Abort()
	respondError(c, http.StatusUnauthorized, constant.CodeUnauthorized, constant.ErrUnauthorized)
}
