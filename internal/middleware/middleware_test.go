package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hyperonym/ratus/internal/middleware"
	"github.com/hyperonym/ratus/internal/reqtest"
)

type group struct{}

func (g *group) Prefixes() []string {
	return []string{"/"}
}

func (g *group) Mount(r *gin.RouterGroup) {
	p := promhttp.Handler()

	r.GET("/prometheus", middleware.Prometheus(), func(c *gin.Context) {
		p.ServeHTTP(c.Writer, c.Request)
	})
}

func TestMiddleware(t *testing.T) {
	h := reqtest.NewHandler(&group{})

	t.Run("prometheus", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/prometheus", nil)
		r1 := reqtest.Record(t, h, req)
		r1.AssertStatusCode(http.StatusOK)
		r2 := reqtest.Record(t, h, req)
		r2.AssertStatusCode(http.StatusOK)
		r2.AssertBodyContains("ratus_request_duration_seconds_bucket")
		r2.AssertBodyContains("endpoint=\"/prometheus\"")
	})
}
