// Package controller implements controllers for API endpoints.
package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/middleware"
)

// @title        Ratus
// @version      v1
// @description  Ratus API Specification

// @contact.name  GitHub
// @contact.url   https://github.com/hyperonym/ratus

// @license.name  Mozilla Public License Version 2.0
// @license.url   https://www.mozilla.org/en-US/MPL/2.0/

// @basePath  /v1

// @tag.name  topics
// @tag.name  tasks
// @tag.name  promises
// @tag.name  health
// @tag.name  metrics

// Middleware instances for binding and normalizing request bodies.
var (
	bindTask    = middleware.Task()
	bindTasks   = middleware.Tasks()
	bindPromise = middleware.Promise()
	bindCommit  = middleware.Commit()
)

// V1 implements endpoint mounting for API version 1.
type V1 struct {
	Pagination gin.HandlerFunc

	Topic   *TopicController
	Task    *TaskController
	Promise *PromiseController
	Health  *HealthController
	Metrics *MetricsController
}

// Prefixes returns the common path prefixes for endpoints in the group.
func (v *V1) Prefixes() []string {
	return []string{"/", "/v1"}
}

// Mount initializes group-level middlewares and mounts the endpoints.
func (v *V1) Mount(r *gin.RouterGroup) {
	r.Use(middleware.Prometheus())

	r.GET("/topics", v.Pagination, v.Topic.GetTopics)
	r.DELETE("/topics", v.Topic.DeleteTopics)

	r.GET("/topics/:topic", v.Topic.GetTopic)
	r.DELETE("/topics/:topic", v.Topic.DeleteTopic)

	r.GET("/topics/:topic/tasks", v.Pagination, v.Task.GetTasks)
	r.POST("/topics/:topic/tasks", bindTasks, v.Task.PostTasks)
	r.PUT("/topics/:topic/tasks", bindTasks, v.Task.PutTasks)
	r.DELETE("/topics/:topic/tasks", v.Task.DeleteTasks)

	r.GET("/topics/:topic/tasks/:id", v.Task.GetTask)
	r.POST("/topics/:topic/tasks/:id", bindTask, v.Task.PostTask)
	r.PUT("/topics/:topic/tasks/:id", bindTask, v.Task.PutTask)
	r.DELETE("/topics/:topic/tasks/:id", v.Task.DeleteTask)
	r.PATCH("/topics/:topic/tasks/:id", bindCommit, v.Task.PatchTask)

	r.GET("/topics/:topic/promises", v.Pagination, v.Promise.GetPromises)
	r.POST("/topics/:topic/promises", bindPromise, v.Promise.PostPromises)
	r.DELETE("/topics/:topic/promises", v.Promise.DeletePromises)

	r.GET("/topics/:topic/promises/:id", v.Promise.GetPromise)
	r.POST("/topics/:topic/promises/:id", bindPromise, v.Promise.PostPromise)
	r.PUT("/topics/:topic/promises/:id", bindPromise, v.Promise.PutPromise)
	r.DELETE("/topics/:topic/promises/:id", v.Promise.DeletePromise)

	r.GET("/healthz", v.Health.GetLiveness)
	r.GET("/livez", v.Health.GetLiveness)
	r.GET("/readyz", v.Health.GetReadiness)

	r.GET("/metrics", v.Metrics.GetMetrics)
}

func send(c *gin.Context, v any, err error) {
	if c.IsAborted() {
		return
	}

	// Create error message and collect server side error.
	if err != nil {
		e := ratus.NewError(err)
		if e.Error.Code >= http.StatusInternalServerError {
			c.Error(err)
		}
		c.AbortWithStatusJSON(e.Error.Code, e)
		return
	}

	// Determine status code for the successful response.
	s := http.StatusOK
	switch x := v.(type) {
	case *ratus.Updated:
		if x.Created > 0 {
			s = http.StatusCreated
		}
	}

	c.JSON(s, v)
}
