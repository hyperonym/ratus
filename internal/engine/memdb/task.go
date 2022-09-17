package memdb

import (
	"context"

	"github.com/hyperonym/ratus"
)

// ListTasks lists all tasks in a topic.
func (g *Engine) ListTasks(ctx context.Context, topic string, limit, offset int) ([]*ratus.Task, error) {
	return nil, nil
}

// InsertTasks inserts a batch of tasks while ignoring existing ones.
func (g *Engine) InsertTasks(ctx context.Context, ts []*ratus.Task) (*ratus.Updated, error) {
	return nil, nil
}

// UpsertTasks inserts or updates a batch of tasks.
func (g *Engine) UpsertTasks(ctx context.Context, ts []*ratus.Task) (*ratus.Updated, error) {
	return nil, nil
}

// DeleteTasks deletes all tasks in a topic.
func (g *Engine) DeleteTasks(ctx context.Context, topic string) (*ratus.Deleted, error) {
	return nil, nil
}

// GetTask gets a task by its unique ID.
func (g *Engine) GetTask(ctx context.Context, id string) (*ratus.Task, error) {
	return nil, nil
}

// InsertTask inserts a new task.
func (g *Engine) InsertTask(ctx context.Context, t *ratus.Task) (*ratus.Updated, error) {
	return nil, nil
}

// UpsertTask inserts or updates a task.
func (g *Engine) UpsertTask(ctx context.Context, t *ratus.Task) (*ratus.Updated, error) {
	return nil, nil
}

// DeleteTask deletes a task by its unique ID.
func (g *Engine) DeleteTask(ctx context.Context, id string) (*ratus.Deleted, error) {
	return nil, nil
}
