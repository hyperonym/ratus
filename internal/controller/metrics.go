package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hyperonym/ratus/internal/engine"
)

// MetricsController provides handlers for metrics-related endpoints.
type MetricsController struct {
	handler http.Handler
}

// NewMetricsController creates a new MetricsController.
func NewMetricsController(g engine.Engine) *MetricsController {
	return &MetricsController{promhttp.Handler()}
}

// GetMetrics gets Prometheus metrics of the instance.
// @summary  Get Prometheus metrics of the instance
// @router   /metrics [get]
// @tags     metrics
// @produce  text/plain
// @success  200 {object} string
func (r *MetricsController) GetMetrics(c *gin.Context) {
	r.handler.ServeHTTP(c.Writer, c.Request)
}
