package docs_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus/docs"
)

func handle(t *testing.T, f http.Handler, req *http.Request) (int, http.Header, []byte) {
	t.Helper()
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	d, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, res.Header, d
}

func TestSwagger(t *testing.T) {
	var g docs.Swagger
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	for _, p := range g.Prefixes() {
		g.Mount(r.Group(p))
	}
	f := r.Handler()

	t.Run("index", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		s, _, b := handle(t, f, req)
		if s != http.StatusOK {
			t.Fail()
		}
		if !bytes.Contains(b, []byte("</head>")) {
			t.Fail()
		}
	})

	t.Run("swagger.json", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/swagger.json", nil)
		s, h, b := handle(t, f, req)
		if s != http.StatusOK {
			t.Fail()
		}
		if !strings.Contains(h.Get("Content-Type"), "application/json") {
			t.Fail()
		}
		if !bytes.Contains(b, []byte("\"swagger\": \"")) {
			t.Fail()
		}
	})

	t.Run("swagger.yaml", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/swagger.yaml", nil)
		s, _, b := handle(t, f, req)
		if s != http.StatusOK {
			t.Fail()
		}
		if !bytes.Contains(b, []byte("swagger: ")) {
			t.Fail()
		}
	})

	t.Run("openapi.json", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
		s, h, b := handle(t, f, req)
		if s != http.StatusOK {
			t.Fail()
		}
		if !strings.Contains(h.Get("Content-Type"), "application/json") {
			t.Fail()
		}
		if !bytes.Contains(b, []byte("\"openapi\": \"")) {
			t.Fail()
		}
	})

	t.Run("openapi.yaml", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
		s, _, b := handle(t, f, req)
		if s != http.StatusOK {
			t.Fail()
		}
		if !bytes.Contains(b, []byte("openapi: ")) {
			t.Fail()
		}
	})

	t.Run("swagger-ui.min.css", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/swagger-ui/swagger-ui.min.css", nil)
		s, h, b := handle(t, f, req)
		if s != http.StatusOK {
			t.Fail()
		}
		if !strings.Contains(h.Get("Content-Type"), "text/css") {
			t.Fail()
		}
		if !bytes.Contains(b, []byte(".swagger-ui")) {
			t.Fail()
		}
	})
}
