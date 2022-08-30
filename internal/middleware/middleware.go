// Package middleware provides functions to inspect and transform requests.
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus"
)

func fail(c *gin.Context, err error) {
	e := ratus.NewError(err)
	c.AbortWithStatusJSON(e.Error.Code, e)
}
