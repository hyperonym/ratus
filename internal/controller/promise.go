package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/engine"
	"github.com/hyperonym/ratus/internal/metrics"
	"github.com/hyperonym/ratus/internal/middleware"
)

// PromiseController provides handlers for promise-related endpoints.
type PromiseController struct {
	Engine engine.Engine
}

// NewPromiseController creates a new PromiseController.
func NewPromiseController(g engine.Engine) *PromiseController {
	return &PromiseController{g}
}

// GetPromises lists all promises in a topic.
// @summary  List all promises in a topic
// @router   /topics/{topic}/promises [get]
// @tags     promises
// @param    topic path string true "Name of the topic"
// @param    limit query int false "Maximum number of resources to return"
// @param    offset query int false "Number of resources to skip"
// @produce  application/json
// @success  200 {object} ratus.Promises
// @failure  400 {object} ratus.Error
// @failure  500 {object} ratus.Error
func (r *PromiseController) GetPromises(c *gin.Context) {
	v, err := r.Engine.ListPromises(c.Request.Context(), c.Param(middleware.ParamTopic), c.GetInt(middleware.ParamLimit), c.GetInt(middleware.ParamOffset))
	send(c, &ratus.Promises{Data: v}, err)
}

// PostPromises claims the next available task in the topic based on the scheduled time.
// @summary  Claim the next available task in the topic based on the scheduled time
// @router   /topics/{topic}/promises [post]
// @tags     promises
// @param    topic path string true "Name of the topic"
// @param    promise body ratus.Promise true "Wildcard promise object to be inserted"
// @accept   application/json
// @produce  application/json
// @success  200 {object} ratus.Task
// @failure  400 {object} ratus.Error
// @failure  404 {object} ratus.Error
// @failure  500 {object} ratus.Error
func (r *PromiseController) PostPromises(c *gin.Context) {
	p := c.MustGet(middleware.ParamPromise).(*ratus.Promise)
	if p.ID != "" {
		r.PostPromise(c)
		return
	}
	v, err := r.Engine.Poll(c.Request.Context(), c.Param(middleware.ParamTopic), p)
	send(c, v, err)
	r.collectMetrics(v)
}

// DeletePromises deletes all promises in a topic.
// @summary  Delete all promises in a topic
// @router   /topics/{topic}/promises [delete]
// @tags     promises
// @param    topic path string true "Name of the topic"
// @produce  application/json
// @success  200 {object} ratus.Deleted
// @failure  500 {object} ratus.Error
func (r *PromiseController) DeletePromises(c *gin.Context) {
	v, err := r.Engine.DeletePromises(c.Request.Context(), c.Param(middleware.ParamTopic))
	send(c, v, err)
}

// GetPromise gets a promise by the unique ID of its target task.
// @summary  Get a promise by the unique ID of its target task
// @router   /topics/{topic}/promises/{id} [get]
// @tags     promises
// @param    topic path string true "Name of the topic"
// @param    id path string true "Unique ID of the target task"
// @produce  application/json
// @success  200 {object} ratus.Promise
// @failure  404 {object} ratus.Error
// @failure  500 {object} ratus.Error
func (r *PromiseController) GetPromise(c *gin.Context) {
	v, err := r.Engine.GetPromise(c.Request.Context(), c.Param(middleware.ParamID))
	send(c, v, err)
}

// PostPromise claims the target task if it is in pending state.
// @summary  Claim the target task if it is in pending state
// @router   /topics/{topic}/promises/{id} [post]
// @tags     promises
// @param    topic path string true "Name of the topic"
// @param    id path string true "Unique ID of the target task"
// @param    promise body ratus.Promise true "Promise object to be inserted"
// @accept   application/json
// @produce  application/json
// @success  200 {object} ratus.Task
// @failure  400 {object} ratus.Error
// @failure  404 {object} ratus.Error
// @failure  409 {object} ratus.Error
// @failure  500 {object} ratus.Error
func (r *PromiseController) PostPromise(c *gin.Context) {
	p := c.MustGet(middleware.ParamPromise).(*ratus.Promise)
	v, err := r.Engine.InsertPromise(c.Request.Context(), p)
	if err == ratus.ErrConflict {
		err = fmt.Errorf("%w: a promise for the same task already exists", err)
	}
	send(c, v, err)
	r.collectMetrics(v)
}

// PutPromise claims the target task regardless of its current state.
// @summary  Claim the target task regardless of its current state
// @router   /topics/{topic}/promises/{id} [put]
// @tags     promises
// @param    topic path string true "Name of the topic"
// @param    id path string true "Unique ID of the target task"
// @param    promise body ratus.Promise true "Promise object to be inserted or updated"
// @accept   application/json
// @produce  application/json
// @success  200 {object} ratus.Task
// @failure  400 {object} ratus.Error
// @failure  404 {object} ratus.Error
// @failure  500 {object} ratus.Error
func (r *PromiseController) PutPromise(c *gin.Context) {
	p := c.MustGet(middleware.ParamPromise).(*ratus.Promise)
	v, err := r.Engine.UpsertPromise(c.Request.Context(), p)
	send(c, v, err)
	r.collectMetrics(v)
}

// DeletePromise deletes a promise by the unique ID of its target task.
// @summary  Delete a promise by the unique ID of its target task
// @router   /topics/{topic}/promises/{id} [delete]
// @tags     promises
// @param    topic path string true "Name of the topic"
// @param    id path string true "Unique ID of the target task"
// @produce  application/json
// @success  200 {object} ratus.Deleted
// @failure  500 {object} ratus.Error
func (r *PromiseController) DeletePromise(c *gin.Context) {
	v, err := r.Engine.DeletePromise(c.Request.Context(), c.Param(middleware.ParamID))
	send(c, v, err)
}

// collectMetrics collects metrics while consuming a task.
func (r *PromiseController) collectMetrics(t *ratus.Task) {

	// Collect task schedule delay.
	if t != nil && t.Scheduled != nil && t.Consumed != nil {
		d := t.Consumed.Sub(*t.Scheduled).Seconds()
		metrics.DelayGauge.WithLabelValues(t.Topic, t.Producer, t.Consumer).Set(d)
	}

	// Collect number of tasks consumed.
	if t != nil {
		metrics.ConsumedCounter.WithLabelValues(t.Topic, t.Producer, t.Consumer).Add(1)
	}
}
