package httpapi

import (
	"fmt"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirojulanam/boilerplate/internal/domain"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Dependencies bundles every handler and port the router needs. Adding a route
// means extending this struct and the composition root, not hiding globals.
type Dependencies struct {
	Health        *HealthHandler
	Auth          *AuthHandler
	TokenVerifier domain.TokenVerifier
	Log           *logrus.Logger
}

// Configure applies process-wide Gin settings. It must be called once from the
// composition root before NewRouter. Keeping this out of NewRouter avoids
// hidden global state in tests and library code.
func Configure(log *logrus.Logger) {
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	gin.DefaultWriter = log.WriterLevel(logrus.InfoLevel)
	gin.DefaultErrorWriter = log.WriterLevel(logrus.ErrorLevel)
}

// NewRouter wires every HTTP route of the service.
func NewRouter(deps Dependencies) *gin.Engine {
	router := gin.New()
	router.HandleMethodNotAllowed = true

	// Order matters: the request ID is available to the logger, and the
	// recovery middleware sits inside the logger so the response log still runs
	// after a recovered panic.
	router.Use(
		withRequestID(),
		withLogging(deps.Log),
		withBodyLimit(constant.MaxRequestBodyBytes),
		gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
			deps.Log.WithFields(logrus.Fields{
				"event":      constant.LogEventProcess,
				"panic":      fmt.Sprint(recovered),
				"stack":      string(debug.Stack()),
				"request_id": c.GetString(constant.RequestIDKey),
			}).Error("panic recovered")
			c.Abort()
			respondError(c, http.StatusInternalServerError, constant.CodeInternalServer, constant.ErrInternalServer)
		}),
	)

	router.GET(constant.HealthPath, deps.Health.Check)

	router.POST(constant.RegisterPath, deps.Auth.Register)
	router.POST(constant.TokenPath, deps.Auth.Token)

	protected := router.Group("")
	protected.Use(withAuthentication(deps.TokenVerifier, deps.Log))
	protected.GET(constant.MePath, deps.Auth.Me)

	router.GET(constant.SwaggerPath, ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, constant.SwaggerUIPath)
	})

	router.NoRoute(func(c *gin.Context) {
		c.Abort()
		respondError(c, http.StatusNotFound, constant.CodeRouteNotFound, constant.ErrRouteNotFound)
	})
	router.NoMethod(func(c *gin.Context) {
		c.Abort()
		respondError(c, http.StatusMethodNotAllowed, constant.CodeMethodNotAllowed, constant.ErrMethodNotAllowed)
	})

	return router
}
