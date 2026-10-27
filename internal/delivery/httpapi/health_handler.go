package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirojulanam/boilerplate/internal/helper"
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
// @Success      200  {object}  HealthResponse
// @Failure      503  {object}  HealthResponse
// @Router       /health [get]
func (h *HealthHandler) Check(c *gin.Context) {
	start := time.Now()
	requestID := c.GetString(constant.RequestIDKey)

	h.log.WithFields(logrus.Fields{
		"event":      constant.LogEventProcess,
		"request_id": requestID,
		"action":     "health",
	}).Info("health check started")

	health := h.health.Check(c.Request.Context())

	// Probe errors stay in the logs; the response only exposes a safe detail.
	for _, check := range health.Checks {
		if check.Err == nil {
			continue
		}
		h.log.WithError(check.Err).WithFields(logrus.Fields{
			"event":      constant.LogEventProcess,
			"request_id": requestID,
			"action":     "health",
			"checker":    check.Name,
		}).Error("health check failed")
	}

	status := http.StatusOK
	if !health.Status.IsHealthy() {
		status = http.StatusServiceUnavailable
	}

	fields := logrus.Fields{
		"event":       constant.LogEventProcess,
		"request_id":  requestID,
		"action":      "health",
		"status":      health.Status,
		"duration_ms": helper.DurationMS(time.Since(start)),
	}
	if health.Status.IsHealthy() {
		h.log.WithFields(fields).Info("health check completed")
	} else {
		h.log.WithFields(fields).Warn("health check completed")
	}

	respondHealth(c, status, health)
}
