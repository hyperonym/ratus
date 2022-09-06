// Package middleware provides functions to inspect and transform requests.
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus"
)

// Name constants for parameter keys.
const (
	ParamID      = "id"
	ParamTopic   = "topic"
	ParamLimit   = "limit"
	ParamOffset  = "offset"
	ParamTask    = "task"
	ParamTasks   = "tasks"
	ParamCommit  = "commit"
	ParamPromise = "promise"
)

func fail(c *gin.Context, err error) {
	e := ratus.NewError(err)
	c.AbortWithStatusJSON(e.Error.Code, e)
}
