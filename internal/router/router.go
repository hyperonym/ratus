// Package router provides API endpoint routing.
package router

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus"
)

// Group defines the interface for mountable API endpoint groups.
// Endpoints in a group share the same path prefix and have common middlewares.
type Group interface {

	// Prefixes returns the common path prefixes for endpoints in the group.
	// A common use of prefixes is for versioning RESTful APIs, and since a
	// group can have multiple path prefixes, the group of the default version
	// can use both the root path and the specific version path.
	Prefixes() []string

	// Mount initializes group-level middlewares and mounts the endpoints.
	Mount(*gin.RouterGroup)
}

// New creates a router engine with all the provided endpoint groups mounted.
func New(groups ...Group) *gin.Engine {

	// Use raw path for matching parameters.
	// Caveat: plus signs '+' in path parameters are unescaped to the space
	// character if the URL path segment contains '%2F' (URL encoded '/').
	// https://github.com/gin-gonic/gin/issues/2633
	r := gin.New()
	r.UseRawPath = true
	r.UnescapePathValues = true

	// Enable logging and profiling if not in release mode.
	// Recommended to collect logs at load balancer level in production.
	if gin.Mode() != gin.ReleaseMode {
		r.Use(gin.Logger())
		pprof.Register(r)
	}

	// Enable CORS for all origins.
	r.Use(cors.Default())

	// Enable gzip with compression level 1 (best speed).
	r.Use(gzip.Gzip(gzip.BestSpeed))

	// Mount endpoints from each group.
	for _, g := range groups {
		for _, p := range g.Prefixes() {
			g.Mount(r.Group(p))
		}
	}

	// Handle 404 not found.
	r.NoRoute(func(c *gin.Context) {
		e := ratus.NewError(ratus.ErrNotFound)
		c.AbortWithStatusJSON(http.StatusNotFound, e)
	})

	return r
}
