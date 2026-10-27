package httpapi

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const (
	// HealthPath is the public health-check route.
	HealthPath = "/api/v1/health"
	// SwaggerPath serves the embedded Swagger UI.
	SwaggerPath = "/swagger/*any"
	// SwaggerUIPath is the Swagger UI entry page.
	SwaggerUIPath = "/swagger/index.html"
)

// NewRouter wires every HTTP route of the service. All output, including Gin's
// own writers, is routed through the provided logrus logger.
func NewRouter(health *HealthHandler, log *logrus.Logger) *gin.Engine {
	if os.Getenv(gin.EnvGinMode) == "" && gin.Mode() == gin.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}

	gin.DefaultWriter = log.WriterLevel(logrus.InfoLevel)
	gin.DefaultErrorWriter = log.WriterLevel(logrus.ErrorLevel)

	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.WithField("panic", fmt.Sprint(recovered)).Error("panic recovered")
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}))
	router.Use(withLogging(log))

	router.GET(HealthPath, health.Check)
	router.GET(SwaggerPath, ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, SwaggerUIPath)
	})

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "route not found"})
	})
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
	})

	return router
}
