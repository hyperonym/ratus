package router_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus/internal/router"
)

type version struct{}

func (v *version) Prefixes() []string {
	return []string{"/", "/v1"}
}

func (v *version) Mount(g *gin.RouterGroup) {
	g.GET("/version", func(c *gin.Context) {
		c.Writer.Write([]byte("42"))
	})
}

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

func TestRouter(t *testing.T) {
	var v version
	f := router.New(&v).Handler()

	t.Run("root", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/version", nil)
		s, _, b := handle(t, f, req)
		if s != http.StatusOK {
			t.Fail()
		}
		if !bytes.Equal(b, []byte("42")) {
			t.Fail()
		}
	})

	t.Run("prefix", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/v1/version", nil)
		s, _, b := handle(t, f, req)
		if s != http.StatusOK {
			t.Fail()
		}
		if !bytes.Equal(b, []byte("42")) {
			t.Fail()
		}
	})

	t.Run("noroute", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/404", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		s, h, _ := handle(t, f, req)
		if s != http.StatusNotFound {
			t.Fail()
		}
		if !strings.Contains(h.Get("Content-Encoding"), "gzip") {
			t.Fail()
		}
	})
}
