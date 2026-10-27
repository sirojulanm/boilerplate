package httpapi

import (
	"fmt"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

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
func NewRouter(health *HealthHandler, log *logrus.Logger) *gin.Engine {
	router := gin.New()
	router.HandleMethodNotAllowed = true

	// Order matters: the request ID is available to the logger, and the
	// recovery middleware sits inside the logger so the access log still runs
	// after a recovered panic.
	router.Use(
		withRequestID(),
		withLogging(log),
		gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
			log.WithFields(logrus.Fields{
				"panic":      fmt.Sprint(recovered),
				"stack":      string(debug.Stack()),
				"request_id": c.GetString(constant.RequestIDKey),
			}).Error("panic recovered")
			c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: constant.ErrInternalServer})
		}),
	)

	router.GET(constant.HealthPath, health.Check)
	router.GET(constant.SwaggerPath, ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, constant.SwaggerUIPath)
	})

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: constant.ErrRouteNotFound})
	})
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, ErrorResponse{Error: constant.ErrMethodNotAllowed})
	})

	return router
}
