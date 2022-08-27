// Package docs embeds documentation files.
package docs

import (
	"embed"
	"net/http"
	"path"

	"github.com/gin-gonic/gin"
)

//go:embed index.html
var index []byte

//go:embed swagger-ui
//go:embed swagger.json swagger.yaml
//go:embed openapi.json openapi.yaml
var swagger embed.FS

// Swagger implements endpoint mounting for API specifications.
type Swagger struct{}

// Prefixes returns the common path prefixes for endpoints in the group.
func (s *Swagger) Prefixes() []string {
	return []string{"/"}
}

// Mount initializes group-level middlewares and mounts the endpoints.
func (s *Swagger) Mount(g *gin.RouterGroup) {
	fs := http.FS(swagger)

	// Serve Swagger UI files.
	g.GET("/", func(c *gin.Context) {
		c.Writer.Write(index)
	})
	g.GET("/swagger-ui/*filepath", func(c *gin.Context) {
		c.FileFromFS(path.Join("/swagger-ui/", c.Param("filepath")), fs)
	})

	// Serve specification files.
	g.GET("/swagger.json", func(c *gin.Context) {
		c.FileFromFS("/swagger.json", fs)
	})
	g.GET("/swagger.yaml", func(c *gin.Context) {
		c.FileFromFS("/swagger.yaml", fs)
	})
	g.GET("/openapi.json", func(c *gin.Context) {
		c.FileFromFS("/openapi.json", fs)
	})
	g.GET("/openapi.yaml", func(c *gin.Context) {
		c.FileFromFS("/openapi.yaml", fs)
	})
}
