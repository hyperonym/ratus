package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/config"
)

// Pagination returns a middleware that normalizes pagination options.
func Pagination(pc *config.PaginationConfig) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Bind query parameters.
		var p struct {
			Limit  int `form:"limit"`
			Offset int `form:"offset"`
		}
		if err := c.ShouldBindQuery(&p); err != nil {
			fail(c, fmt.Errorf("%w: invalid pagination parameters", ratus.ErrBadRequest))
			return
		}

		// The hard-coded default might be greater than the maximum limit,
		// always use the smaller of the two numbers as the default limit.
		if p.Limit == 0 {
			if ratus.DefaultLimit < pc.MaxLimit {
				p.Limit = ratus.DefaultLimit
			} else {
				p.Limit = pc.MaxLimit
			}
		}

		// Validate ranges of limit and offset.
		if p.Limit < 0 {
			fail(c, fmt.Errorf("%w: limit must not be negative", ratus.ErrBadRequest))
			return
		}
		if p.Offset < 0 {
			fail(c, fmt.Errorf("%w: offset must not be negative", ratus.ErrBadRequest))
			return
		}
		if p.Limit > pc.MaxLimit {
			fail(c, fmt.Errorf("%w: exceeded maximum allowed limit of %d", ratus.ErrBadRequest, pc.MaxLimit))
			return
		}
		if p.Offset > pc.MaxOffset {
			fail(c, fmt.Errorf("%w: exceeded maximum allowed offset of %d", ratus.ErrBadRequest, pc.MaxOffset))
			return
		}

		// Store normalized pagination options in the request context.
		c.Set("limit", p.Limit)
		c.Set("offset", p.Offset)

		c.Next()
	}
}
