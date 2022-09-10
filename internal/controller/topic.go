package controller

import (
	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/engine"
	"github.com/hyperonym/ratus/internal/middleware"
)

// TopicController implements handlers for topic-related endpoints.
type TopicController struct {
	Engine engine.Engine
}

// NewTopicController creates a new TopicController.
func NewTopicController(g engine.Engine) *TopicController {
	return &TopicController{g}
}

// GetTopics lists all topics.
// @summary  List all topics
// @router   /topics [get]
// @tags     topics
// @param    limit query int false "Maximum number of resources to return"
// @param    offset query int false "Number of resources to skip"
// @produce  application/json
// @success  200 {object} ratus.Topics
// @failure  400 {object} ratus.Error
// @failure  500 {object} ratus.Error
func (r *TopicController) GetTopics(c *gin.Context) {
	v, err := r.Engine.ListTopics(c.Request.Context(), c.GetInt(middleware.ParamLimit), c.GetInt(middleware.ParamOffset))
	send(c, &ratus.Topics{Data: v}, err)
}

// DeleteTopics deletes all topics and tasks.
// @summary  Delete all topics and tasks
// @router   /topics [delete]
// @tags     topics
// @produce  application/json
// @success  200 {object} ratus.Deleted
// @failure  500 {object} ratus.Error
func (r *TopicController) DeleteTopics(c *gin.Context) {
	v, err := r.Engine.DeleteTopics(c.Request.Context())
	send(c, v, err)
}

// GetTopic gets information about a topic.
// @summary  Get information about a topic
// @router   /topics/{topic} [get]
// @tags     topics
// @param    topic path string true "Name of the topic"
// @produce  application/json
// @success  200 {object} ratus.Topic
// @failure  404 {object} ratus.Error
// @failure  500 {object} ratus.Error
func (r *TopicController) GetTopic(c *gin.Context) {
	v, err := r.Engine.GetTopic(c.Request.Context(), c.Param(middleware.ParamTopic))
	send(c, v, err)
}

// DeleteTopic deletes a topic and its tasks.
// @summary  Delete a topic and its tasks
// @router   /topics/{topic} [delete]
// @tags     topics
// @param    topic path string true "Name of the topic"
// @produce  application/json
// @success  200 {object} ratus.Deleted
// @failure  500 {object} ratus.Error
func (r *TopicController) DeleteTopic(c *gin.Context) {
	v, err := r.Engine.DeleteTopic(c.Request.Context(), c.Param(middleware.ParamTopic))
	send(c, v, err)
}
