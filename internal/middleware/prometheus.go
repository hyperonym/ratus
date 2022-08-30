package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus/internal/metrics"
)

// Prometheus returns a middleware that collects request-related metrics.
func Prometheus() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		c.Next()
		d := time.Since(t).Seconds()
		s := strconv.Itoa(c.Writer.Status())
		metrics.RequestHistogram.WithLabelValues(c.Param("topic"), c.FullPath(), s).Observe(d)
	}
}
