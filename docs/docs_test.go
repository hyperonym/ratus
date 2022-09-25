package docs_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus/docs"
	"github.com/hyperonym/ratus/internal/reqtest"
)

func TestSwagger(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	h := reqtest.NewHandler(&docs.Swagger{})

	t.Run("index", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		r := reqtest.Record(t, h, req)
		r.AssertStatusCode(http.StatusOK)
		r.AssertBodyContains("</head>")
	})

	t.Run("swagger.json", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/swagger.json", nil)
		r := reqtest.Record(t, h, req)
		r.AssertStatusCode(http.StatusOK)
		r.AssertHeaderContains("Content-Type", "application/json")
		r.AssertBodyContains(`"swagger":`)
	})

	t.Run("swagger.yaml", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/swagger.yaml", nil)
		r := reqtest.Record(t, h, req)
		r.AssertStatusCode(http.StatusOK)
		r.AssertBodyContains("swagger: ")
	})

	t.Run("openapi.json", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
		r := reqtest.Record(t, h, req)
		r.AssertStatusCode(http.StatusOK)
		r.AssertHeaderContains("Content-Type", "application/json")
		r.AssertBodyContains(`"openapi":`)
	})

	t.Run("openapi.yaml", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
		r := reqtest.Record(t, h, req)
		r.AssertStatusCode(http.StatusOK)
		r.AssertBodyContains("openapi: ")
	})

	t.Run("swagger-ui.min.css", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/swagger-ui/swagger-ui.min.css", nil)
		r := reqtest.Record(t, h, req)
		r.AssertStatusCode(http.StatusOK)
		r.AssertHeaderContains("Content-Type", "text/css")
		r.AssertBodyContains(".swagger-ui")
	})
}
