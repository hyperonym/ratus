package memdb

import (
	"context"

	"github.com/hyperonym/ratus"
)

// ListTopics lists all topics.
func (g *Engine) ListTopics(ctx context.Context, limit, offset int) ([]*ratus.Topic, error) {
	return nil, nil
}

// DeleteTopics deletes all topics and tasks.
func (g *Engine) DeleteTopics(ctx context.Context) (*ratus.Deleted, error) {
	return nil, nil
}

// GetTopic gets information about a topic.
func (g *Engine) GetTopic(ctx context.Context, topic string) (*ratus.Topic, error) {
	return nil, nil
}

// DeleteTopic deletes a topic and its tasks.
func (g *Engine) DeleteTopic(ctx context.Context, topic string) (*ratus.Deleted, error) {
	return nil, nil
}
