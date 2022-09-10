package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus/internal/engine"
)

// HealthController implements handlers for health-related endpoints.
type HealthController struct {
	Engine engine.Engine
}

// NewHealthController creates a new HealthController.
func NewHealthController(g engine.Engine) *HealthController {
	return &HealthController{g}
}

// GetLiveness checks the liveness of the instance.
// @summary  Check the liveness of the instance
// @router   /livez [get]
// @tags     health
// @success  200
func (r *HealthController) GetLiveness(c *gin.Context) {
	c.Status(http.StatusOK)
}

// GetReadiness checks the readiness of the instance.
// @summary  Check the readiness of the instance
// @router   /readyz [get]
// @tags     health
// @success  200
// @failure  503 {object} ratus.Error
func (r *HealthController) GetReadiness(c *gin.Context) {
	if err := r.Engine.Ready(c.Request.Context()); err != nil {
		send(c, nil, err)
		return
	}
	c.Status(http.StatusOK)
}
