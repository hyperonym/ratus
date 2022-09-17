package memdb

import (
	"context"

	"github.com/hyperonym/ratus"
)

// Chore recovers timed out tasks and deletes expired tasks.
func (g *Engine) Chore(ctx context.Context) error {
	return nil
}

// Poll makes a promise to claim and execute the next available task in a topic.
func (g *Engine) Poll(ctx context.Context, topic string, p *ratus.Promise) (*ratus.Task, error) {
	return nil, nil
}

// Commit applies a set of updates to a task and returns the updated task.
func (g *Engine) Commit(ctx context.Context, id string, m *ratus.Commit) (*ratus.Task, error) {
	return nil, nil
}
