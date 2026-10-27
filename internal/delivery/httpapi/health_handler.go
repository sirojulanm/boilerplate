package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirojulanam/boilerplate/internal/usecase"
)

// HealthHandler exposes the health usecase over HTTP using Gin.
type HealthHandler struct {
	usecase usecase.Health
}

// NewHealthHandler wires a health usecase into the HTTP layer.
func NewHealthHandler(uc usecase.Health) *HealthHandler {
	return &HealthHandler{usecase: uc}
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
	health := h.usecase.Check(c.Request.Context())

	status := http.StatusOK
	if !health.Status.IsHealthy() {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, health)
}
