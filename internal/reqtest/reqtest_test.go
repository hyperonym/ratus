package reqtest_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus/internal/reqtest"
)

func TestRequest(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	h := reqtest.NewHandler(&reqtest.StubGroup{})

	t.Run("get", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/version", nil)
		r := reqtest.Record(t, h, req)
		r.AssertStatusCode(http.StatusOK)
		r.AssertHeaderContains("Content-Type", "text/plain")
		r.AssertBodyContains("42")
	})

	t.Run("post", func(t *testing.T) {
		t.Parallel()
		req := reqtest.NewRequestJSON(http.MethodPost, "/echo", gin.H{"foo": "bar"})
		r := reqtest.Record(t, h, req)
		r.AssertStatusCode(http.StatusOK)
		r.AssertHeaderContains("Content-Type", "application/json")
		r.AssertBodyContains(`{"foo":"bar"}`)
	})
}
