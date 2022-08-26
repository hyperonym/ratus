// Package router provides API endpoint routing.
package router

import "github.com/gin-gonic/gin"

// Group defines the interface for mountable API endpoint groups.
// Endpoints in a group share the same path prefix and have common middlewares.
type Group interface {

	// Prefixes returns the common path prefixes for endpoints in the group.
	// A common use of prefixes is for versioning RESTful APIs, and since a
	// group can have multiple path prefixes, the group of the default version
	// can use both the root path and the specific version path.
	Prefixes() []string

	// Mount initializes group-level middlewares and mounts the endpoints.
	Mount(g *gin.RouterGroup)
}
