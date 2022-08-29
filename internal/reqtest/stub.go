package reqtest

import (
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
