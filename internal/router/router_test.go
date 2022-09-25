package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus/internal/reqtest"
	"github.com/hyperonym/ratus/internal/router"
)

func TestRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := router.New(&reqtest.StubGroup{}).Handler()

	t.Run("root", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/version", nil)
		r := reqtest.Record(t, h, req)
		r.AssertStatusCode(http.StatusOK)
		r.AssertBodyContains("42")
	})

	t.Run("prefix", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/stub/version", nil)
		r := reqtest.Record(t, h, req)
		r.AssertStatusCode(http.StatusOK)
		r.AssertBodyContains("42")
	})

	t.Run("noroute", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/404", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		r := reqtest.Record(t, h, req)
		r.AssertStatusCode(http.StatusNotFound)
		r.AssertHeaderContains("Content-Encoding", "gzip")
	})
}
