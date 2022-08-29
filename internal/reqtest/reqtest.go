// Package reqtest provides utilities for testing requests and responses.
package reqtest

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

// ResponseRecord contains the status code, header, and body of the recorded
// response. More fields may be populated in the future, so callers should not
// DeepEqual the result in tests. Use the provided assertion methods instead.
type ResponseRecord struct {
	StatusCode int
	Header     http.Header
	Body       []byte

	// Keep a reference to the test/subtest for assertion methods.
	test *testing.T
}

// AssertStatusCode marks the test as failed if the response status code does
// not match the expected value.
func (r *ResponseRecord) AssertStatusCode(v int) {
	r.test.Helper()
	if r.StatusCode != v {
		r.test.Errorf("incorrect response status code, expected %d, got %d", v, r.StatusCode)
	}
}

// AssertHeaderContains marks the test as failed if the response header field
// does not contain the provided substring.
func (r *ResponseRecord) AssertHeaderContains(field string, v string) {
	r.test.Helper()
	if h := r.Header.Get(field); !strings.Contains(h, v) {
		r.test.Errorf("response header field %q does not contain %q, got %q", field, v, h)
	}
}

// AssertBodyContains marks the test as failed if the response body does not
// contain the provided substring.
func (r *ResponseRecord) AssertBodyContains(v string) {
	r.test.Helper()
	if !bytes.Contains(r.Body, []byte(v)) {
		r.test.Errorf("response body does not contain %q, got %q", v, string(r.Body))
	}
}

// Record the response generated by the handler for testing.
func Record(t *testing.T, h http.Handler, req *http.Request) *ResponseRecord {
	t.Helper()

	// Handle the request and write the response to a recorder.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	// Read and save the response body as bytes.
	d, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	return &ResponseRecord{
		StatusCode: res.StatusCode,
		Header:     res.Header,
		Body:       d,
		test:       t,
	}
}

// StubGroup is an endpoint group that returns hard-coded values for testing.
type StubGroup struct{}

// Prefixes returns the common path prefixes for endpoints in the group.
func (s *StubGroup) Prefixes() []string {
	return []string{"/", "/stub"}
}

// Mount initializes group-level middlewares and mounts the endpoints.
func (s *StubGroup) Mount(g *gin.RouterGroup) {
	g.GET("/version", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain")
		c.String(http.StatusOK, "42")
	})
}

// NewHandler creates a handler from an endpoint group for testing.
func NewHandler(g router.Group) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	for _, p := range g.Prefixes() {
		g.Mount(r.Group(p))
	}
	return r.Handler()
}
