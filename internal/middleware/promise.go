package middleware

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus"
)

// Promise returns a middleware that normalizes promises in request bodies.
func Promise() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Promise is a relatively simple data structure that can be submitted
		// either through the request body or query parameters.
		var p ratus.Promise
		c.ShouldBindJSON(&p)
		c.ShouldBindQuery(&p)

		// Validate and normalize the promise.
		if err := normalizePromise(&p, c.Param(ParamID)); err != nil {
			fail(c, fmt.Errorf("%w: %v", ratus.ErrBadRequest, err))
			return
		}

		// Store the normalized promise in the request context.
		c.Set(ParamPromise, &p)

		c.Next()
	}
}

func normalizePromise(p *ratus.Promise, id string) error {

	// Normalize and validate ID.
	if id != "" && p.ID == "" {
		p.ID = id
	}
	if id != "" && p.ID != id {
		return errors.New("promise ID is inconsistent with the path parameter")
	}

	// Normalize deadline time.
	if p.Deadline == nil {
		if p.Timeout == "" {
			p.Timeout = ratus.DefaultTimeout
		}
		d, err := time.ParseDuration(p.Timeout)
		if err != nil {
			return err
		}
		n := time.Now().Add(d)
		p.Deadline = &n
	}

	// Clear the timeout field after converting to an absolute timestamp.
	p.Timeout = ""

	return nil
}
