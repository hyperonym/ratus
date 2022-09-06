package middleware

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus"
)

// Task returns a middleware that normalizes tasks in request bodies.
func Task() gin.HandlerFunc {
	return func(c *gin.Context) {

		// The request body must not be empty and contains a valid task.
		var t ratus.Task
		if err := c.ShouldBindJSON(&t); err != nil {
			if err == io.EOF {
				fail(c, fmt.Errorf("%w: missing request body", ratus.ErrBadRequest))
				return
			}
			fail(c, fmt.Errorf("%w: %v", ratus.ErrBadRequest, err))
			return
		}

		// Validate and normalize the task.
		if err := normalizeTask(&t, c.Param(ParamID), c.Param(ParamTopic)); err != nil {
			fail(c, fmt.Errorf("%w: %v", ratus.ErrBadRequest, err))
			return
		}

		// Store the normalized task in the request context.
		c.Set(ParamTask, &t)

		c.Next()
	}
}

// Tasks returns a middleware that normalizes task lists in request bodies.
func Tasks() gin.HandlerFunc {
	return func(c *gin.Context) {

		// The request body must not be empty and contains a valid task list.
		var ts ratus.Tasks
		if err := c.ShouldBindJSON(&ts); err != nil {
			if err == io.EOF {
				fail(c, fmt.Errorf("%w: missing request body", ratus.ErrBadRequest))
				return
			}
			fail(c, fmt.Errorf("%w: %v", ratus.ErrBadRequest, err))
			return
		}

		// Validate and normalize all tasks in the list.
		p := c.Param(ParamTopic)
		for _, t := range ts.Data {
			if err := normalizeTask(t, "", p); err != nil {
				fail(c, fmt.Errorf("%w: %v", ratus.ErrBadRequest, err))
				return
			}
		}

		// Allow task lists to be empty.
		if ts.Data == nil {
			ts.Data = make([]*ratus.Task, 0)
		}

		// Store the normalized task list in the request context.
		c.Set(ParamTasks, &ts)

		c.Next()
	}
}

func normalizeTask(t *ratus.Task, id, topic string) error {

	// Normalize and validate ID.
	if t.ID == "" {
		t.ID = id
	}
	if t.ID == "" {
		return errors.New("task ID must not be empty")
	}
	if id != "" && t.ID != id {
		return errors.New("task ID is inconsistent with the path parameter")
	}

	// Normalize and validate topic.
	if t.Topic == "" && topic != "" {
		t.Topic = topic
	}
	if t.Topic == "" {
		return errors.New("topic must not be empty")
	}

	// Validate task state.
	if t.State < ratus.TaskStatePending || t.State > ratus.TaskStateArchived {
		return fmt.Errorf("invalid state %d", t.State)
	}

	// Normalize produced time.
	n := time.Now()
	if t.Produced == nil {
		t.Produced = &n
	}

	// Normalize scheduled time.
	if t.Defer != "" && t.Scheduled == nil {
		d, err := time.ParseDuration(t.Defer)
		if err != nil {
			return err
		}
		s := n.Add(d)
		t.Scheduled = &s
	}
	if t.Scheduled == nil {
		t.Scheduled = &n
	}

	// Clear the defer field after converting to an absolute timestamp.
	t.Defer = ""

	return nil
}
