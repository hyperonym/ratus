package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hyperonym/ratus/internal/config"
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

	r.GET("/pagination/20", middleware.Pagination(&config.PaginationConfig{
		MaxLimit:  20,
		MaxOffset: 20,
	}), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"limit":  c.GetInt("limit"),
			"offset": c.GetInt("offset"),
		})
	})

	r.GET("/pagination/5", middleware.Pagination(&config.PaginationConfig{
		MaxLimit:  5,
		MaxOffset: 5,
	}), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"limit":  c.GetInt("limit"),
			"offset": c.GetInt("offset"),
		})
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

	t.Run("pagination", func(t *testing.T) {
		t.Parallel()

		t.Run("20", func(t *testing.T) {
			t.Parallel()

			t.Run("default", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/pagination/20?offset=5", nil)
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusOK)
				r.AssertBodyContains("\"limit\":10")
				r.AssertBodyContains("\"offset\":5")
			})

			t.Run("bind", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/pagination/20?offset=foo", nil)
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusBadRequest)
				r.AssertBodyContains("invalid pagination parameters")
			})

			t.Run("limit", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/pagination/20?limit=21", nil)
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusBadRequest)
				r.AssertBodyContains("exceeded maximum allowed limit of 20")
			})

			t.Run("offset", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/pagination/20?offset=21", nil)
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusBadRequest)
				r.AssertBodyContains("exceeded maximum allowed offset of 20")
			})
		})

		t.Run("5", func(t *testing.T) {
			t.Parallel()

			t.Run("default", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/pagination/5", nil)
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusOK)
				r.AssertBodyContains("\"limit\":5")
				r.AssertBodyContains("\"offset\":0")
			})

			t.Run("limit", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/pagination/5?limit=-1", nil)
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusBadRequest)
				r.AssertBodyContains("limit must not be negative")
			})

			t.Run("offset", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/pagination/5?offset=-1", nil)
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusBadRequest)
				r.AssertBodyContains("offset must not be negative")
			})
		})
	})
}
