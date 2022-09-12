package reqtest

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus/internal/router"
)

// StubGroup is an endpoint group that returns hard-coded values for testing.
type StubGroup struct{}

// Prefixes returns the common path prefixes for endpoints in the group.
func (s *StubGroup) Prefixes() []string {
	return []string{"/", "/stub"}
}

// Mount initializes group-level middlewares and mounts the endpoints.
func (s *StubGroup) Mount(r *gin.RouterGroup) {
	r.GET("/version", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain")
		c.String(http.StatusOK, "42")
	})
	r.POST("/echo", func(c *gin.Context) {
		if t := c.Request.Header.Get("Content-Type"); t != "" {
			c.Header("Content-Type", t)
		}
		io.Copy(c.Writer, c.Request.Body)
	})
}

// NewHandler creates a handler from an endpoint group for testing.
func NewHandler(g router.Group) http.Handler {
	r := gin.New()
	for _, p := range g.Prefixes() {
		g.Mount(r.Group(p))
	}
	return r.Handler()
}
