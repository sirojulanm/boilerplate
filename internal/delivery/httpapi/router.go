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

// NewRouter wires every HTTP route of the service. All output, including Gin's
// own writers, is routed through the provided logrus logger.
func NewRouter(health *HealthHandler, log *logrus.Logger) *gin.Engine {
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	gin.DefaultWriter = log.WriterLevel(logrus.InfoLevel)
	gin.DefaultErrorWriter = log.WriterLevel(logrus.ErrorLevel)

	router := gin.New()
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.WithField("panic", fmt.Sprint(recovered)).Error("panic recovered")
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}))
	router.Use(withLogging(log))

	router.GET("/api/v1/health", health.Check)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/swagger/index.html")
	})

	return router
}
