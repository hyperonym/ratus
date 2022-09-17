package memdb

import (
	"context"

	"github.com/hyperonym/ratus"
)

// ListPromises lists all promises in a topic.
func (g *Engine) ListPromises(ctx context.Context, topic string, limit, offset int) ([]*ratus.Promise, error) {
	return nil, nil
}

// DeletePromises deletes all promises in a topic.
func (g *Engine) DeletePromises(ctx context.Context, topic string) (*ratus.Deleted, error) {
	return nil, nil
}

// GetPromise gets a promise by the unique ID of its target task.
func (g *Engine) GetPromise(ctx context.Context, id string) (*ratus.Promise, error) {
	return nil, nil
}

// InsertPromise makes a promise to claim and execute a task if it is in pending state.
func (g *Engine) InsertPromise(ctx context.Context, p *ratus.Promise) (*ratus.Task, error) {
	return nil, nil
}

// UpsertPromise makes a promise to claim and execute a task regardless of its current state.
func (g *Engine) UpsertPromise(ctx context.Context, p *ratus.Promise) (*ratus.Task, error) {
	return nil, nil
}

// DeletePromise deletes a promise by the unique ID of its target task.
func (g *Engine) DeletePromise(ctx context.Context, id string) (*ratus.Deleted, error) {
	return nil, nil
}
