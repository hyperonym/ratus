package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus"
)

// Commit returns a middleware that normalizes commits in request bodies.
func Commit() gin.HandlerFunc {
	return func(c *gin.Context) {

		// All fields are optional in a commit.
		// An empty commit sets the state of the task to "completed".
		var m ratus.Commit
		c.ShouldBindJSON(&m)

		// Validate and normalize the commit.
		if err := normalizeCommit(&m); err != nil {
			fail(c, fmt.Errorf("%w: %v", ratus.ErrBadRequest, err))
			return
		}

		// Store the normalized commit in the request context.
		c.Set("commit", &m)

		c.Next()
	}
}

func normalizeCommit(m *ratus.Commit) error {

	// Normalize and validate state.
	if m.State == nil {
		s := ratus.TaskStateCompleted
		m.State = &s
	}
	if *m.State < ratus.TaskStatePending || *m.State > ratus.TaskStateArchived {
		return fmt.Errorf("invalid target state %d", *m.State)
	}

	// Normalize scheduled time.
	if m.Defer != "" && m.Scheduled == nil {
		d, err := time.ParseDuration(m.Defer)
		if err != nil {
			return err
		}
		n := time.Now().Add(d)
		m.Scheduled = &n
	}

	// Clear the defer field after converting to an absolute timestamp.
	m.Defer = ""

	return nil
}
