// Package docs embeds documentation files.
package docs

import (
	_ "embed"
	"encoding/json"

	"github.com/gin-gonic/gin"
)

//go:embed openapi.json
var openapi []byte

//go:embed swagger.json
var swagger []byte

// OpenAPI 3.0 specification.
var OpenAPI gin.H

// Swagger 2.0 specification.
var Swagger gin.H

// Parse embedded files.
func init() {
	if err := json.Unmarshal(openapi, &OpenAPI); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(swagger, &Swagger); err != nil {
		panic(err)
	}
}
