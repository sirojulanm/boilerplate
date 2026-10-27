package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirojulanam/boilerplate/internal/usecase"
	"github.com/sirupsen/logrus"
)

// HealthHandler exposes the health usecase over HTTP using Gin.
type HealthHandler struct {
	health usecase.Health
	log    *logrus.Logger
}

// NewHealthHandler wires a health usecase into the HTTP layer.
func NewHealthHandler(uc usecase.Health, log *logrus.Logger) *HealthHandler {
	return &HealthHandler{health: uc, log: log}
}

// Check godoc
// @Summary      Health check
// @Description  Returns the current health of the service and its dependencies.
// @Tags         health
// @Produce      json
// @Success      200  {object}  domain.Health
// @Failure      503  {object}  domain.Health
// @Router       /health [get]
func (h *HealthHandler) Check(c *gin.Context) {
	health := h.health.Check(c.Request.Context())

	// Probe errors stay in the logs; the response only exposes a safe detail.
	for _, check := range health.Checks {
		if check.Err == nil {
			continue
		}
		h.log.WithError(check.Err).WithFields(logrus.Fields{
			"checker":    check.Name,
			"request_id": c.GetString(constant.RequestIDKey),
		}).Error("health check failed")
	}

	status := http.StatusOK
	if !health.Status.IsHealthy() {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, health)
}
