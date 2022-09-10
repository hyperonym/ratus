package controller

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/engine"
	"github.com/hyperonym/ratus/internal/metrics"
	"github.com/hyperonym/ratus/internal/middleware"
)

// TaskController implements handlers for task-related endpoints.
type TaskController struct {
	Engine engine.Engine
}

// NewTaskController creates a new TaskController.
func NewTaskController(g engine.Engine) *TaskController {
	return &TaskController{g}
}

// GetTasks lists all tasks in a topic.
// @summary  List all tasks in a topic
// @router   /topics/{topic}/tasks [get]
// @tags     tasks
// @param    topic path string true "Name of the topic"
// @param    limit query int false "Maximum number of resources to return"
// @param    offset query int false "Number of resources to skip"
// @produce  application/json
// @success  200 {object} ratus.Tasks
// @failure  400 {object} ratus.Error
// @failure  500 {object} ratus.Error
func (r *TaskController) GetTasks(c *gin.Context) {
	v, err := r.Engine.ListTasks(c.Request.Context(), c.Param(middleware.ParamTopic), c.GetInt(middleware.ParamLimit), c.GetInt(middleware.ParamOffset))
	send(c, &ratus.Tasks{Data: v}, err)
}

// PostTasks inserts a batch of tasks while ignoring existing ones.
// @summary  Insert a batch of tasks while ignoring existing ones
// @router   /topics/{topic}/tasks [post]
// @tags     tasks
// @param    topic path string true "Name of the topic"
// @param    tasks body ratus.Tasks true "Batch of tasks to be inserted"
// @accept   application/json
// @produce  application/json
// @success  200 {object} ratus.Updated
// @success  201 {object} ratus.Updated
// @failure  400 {object} ratus.Error
// @failure  500 {object} ratus.Error
func (r *TaskController) PostTasks(c *gin.Context) {
	ts := c.MustGet(middleware.ParamTasks).(*ratus.Tasks)
	v, err := r.Engine.InsertTasks(c.Request.Context(), ts.Data)
	send(c, v, err)

	// Collect number of tasks produced.
	if v != nil && v.Created > 0 && len(ts.Data) > 0 {
		metrics.ProducedCounter.WithLabelValues(c.Param(middleware.ParamTopic), ts.Data[0].Producer).Add(float64(v.Created))
	}
}

// PutTasks inserts or updates a batch of tasks.
// @summary  Insert or update a batch of tasks
// @router   /topics/{topic}/tasks [put]
// @tags     tasks
// @param    topic path string true "Name of the topic"
// @param    tasks body ratus.Tasks true "Batch of tasks to be inserted or updated"
// @accept   application/json
// @produce  application/json
// @success  200 {object} ratus.Updated
// @success  201 {object} ratus.Updated
// @failure  400 {object} ratus.Error
// @failure  500 {object} ratus.Error
func (r *TaskController) PutTasks(c *gin.Context) {
	ts := c.MustGet(middleware.ParamTasks).(*ratus.Tasks)
	v, err := r.Engine.UpsertTasks(c.Request.Context(), ts.Data)
	send(c, v, err)

	// Collect number of tasks produced.
	if v != nil && v.Created+v.Updated > 0 && len(ts.Data) > 0 {
		metrics.ProducedCounter.WithLabelValues(c.Param(middleware.ParamTopic), ts.Data[0].Producer).Add(float64(v.Created + v.Updated))
	}
}

// DeleteTasks deletes all tasks in a topic.
// @summary  Delete all tasks in a topic
// @router   /topics/{topic}/tasks [delete]
// @tags     tasks
// @param    topic path string true "Name of the topic"
// @produce  application/json
// @success  200 {object} ratus.Deleted
// @failure  500 {object} ratus.Error
func (r *TaskController) DeleteTasks(c *gin.Context) {
	v, err := r.Engine.DeleteTasks(c.Request.Context(), c.Param(middleware.ParamTopic))
	send(c, v, err)
}

// GetTask gets a task by its unique ID.
// @summary  Get a task by its unique ID
// @router   /topics/{topic}/tasks/{id} [get]
// @tags     tasks
// @param    topic path string true "Name of the topic"
// @param    id path string true "Unique ID of the task"
// @produce  application/json
// @success  200 {object} ratus.Task
// @failure  404 {object} ratus.Error
// @failure  500 {object} ratus.Error
func (r *TaskController) GetTask(c *gin.Context) {
	v, err := r.Engine.GetTask(c.Request.Context(), c.Param(middleware.ParamID))
	send(c, v, err)
}

// PostTask inserts a new task.
// @summary  Insert a new task
// @router   /topics/{topic}/tasks/{id} [post]
// @tags     tasks
// @param    topic path string true "Name of the topic"
// @param    id path string true "Unique ID of the task"
// @param    task body ratus.Task true "Task object to be inserted"
// @accept   application/json
// @produce  application/json
// @success  201 {object} ratus.Updated
// @failure  400 {object} ratus.Error
// @failure  409 {object} ratus.Error
// @failure  500 {object} ratus.Error
func (r *TaskController) PostTask(c *gin.Context) {
	t := c.MustGet(middleware.ParamTask).(*ratus.Task)
	v, err := r.Engine.InsertTask(c.Request.Context(), t)
	if err == ratus.ErrConflict {
		err = fmt.Errorf("%w: a task with the same ID already exists", err)
	}
	send(c, v, err)

	// Collect number of tasks produced.
	if v != nil && v.Created > 0 {
		metrics.ProducedCounter.WithLabelValues(t.Topic, t.Producer).Add(float64(v.Created))
	}
}

// PutTask inserts or updates a task.
// @summary  Insert or update a task
// @router   /topics/{topic}/tasks/{id} [put]
// @tags     tasks
// @param    topic path string true "Name of the topic"
// @param    id path string true "Unique ID of the task"
// @param    task body ratus.Task true "Task object to be inserted or updated"
// @accept   application/json
// @produce  application/json
// @success  200 {object} ratus.Updated
// @success  201 {object} ratus.Updated
// @failure  400 {object} ratus.Error
// @failure  500 {object} ratus.Error
func (r *TaskController) PutTask(c *gin.Context) {
	t := c.MustGet(middleware.ParamTask).(*ratus.Task)
	v, err := r.Engine.UpsertTask(c.Request.Context(), t)
	send(c, v, err)

	// Collect number of tasks produced.
	if v != nil && v.Created+v.Updated > 0 {
		metrics.ProducedCounter.WithLabelValues(t.Topic, t.Producer).Add(float64(v.Created + v.Updated))
	}
}

// DeleteTask deletes a task by its unique ID.
// @summary  Delete a task by its unique ID
// @router   /topics/{topic}/tasks/{id} [delete]
// @tags     tasks
// @param    topic path string true "Name of the topic"
// @param    id path string true "Unique ID of the task"
// @produce  application/json
// @success  200 {object} ratus.Deleted
// @failure  500 {object} ratus.Error
func (r *TaskController) DeleteTask(c *gin.Context) {
	v, err := r.Engine.DeleteTask(c.Request.Context(), c.Param(middleware.ParamID))
	send(c, v, err)
}

// PatchTask applies a set of updates to a task and returns the updated task.
// @summary  Apply a set of updates to a task and return the updated task
// @router   /topics/{topic}/tasks/{id} [patch]
// @tags     tasks
// @param    topic path string true "Name of the topic"
// @param    id path string true "Unique ID of the task"
// @param    commit body ratus.Commit false "Commit object to be applied"
// @accept   application/json
// @produce  application/json
// @success  200 {object} ratus.Task
// @failure  400 {object} ratus.Error
// @failure  404 {object} ratus.Error
// @failure  409 {object} ratus.Error
// @failure  500 {object} ratus.Error
func (r *TaskController) PatchTask(c *gin.Context) {
	m := c.MustGet(middleware.ParamCommit).(*ratus.Commit)
	v, err := r.Engine.Commit(c.Request.Context(), c.Param(middleware.ParamID), m)
	if err == ratus.ErrConflict {
		err = fmt.Errorf("%w: the task may have been modified by others", err)
	}
	send(c, v, err)

	// Collect task execution time.
	if v != nil && v.Consumed != nil {
		d := time.Since(*v.Consumed).Seconds()
		metrics.ExecutionGauge.WithLabelValues(v.Topic, v.Producer, v.Consumer).Set(d)
	}

	// Collect number of tasks committed.
	if v != nil {
		metrics.CommittedCounter.WithLabelValues(v.Topic, v.Producer, v.Consumer).Add(1)
	}
}
