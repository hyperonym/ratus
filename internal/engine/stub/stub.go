// Package stub provides a stub engine that returns canned data for testing.
package stub

import (
	"context"
	"time"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/nonce"
)

// Canned data for testing.
var (
	cannedID      = "id"
	cannedTopic   = "topic"
	cannedDate    = time.Date(2022, time.July, 29, 20, 0, 0, 0, time.UTC)
	cannedPayload = "payload"
)

// Engine is a stub engine that returns canned data for testing.
type Engine struct {
	Err error
}

// Open or connect to the storage engine.
func (g *Engine) Open(ctx context.Context) error {
	return g.Err
}

// Close or disconnect from the storage engine.
func (g *Engine) Close(ctx context.Context) error {
	return g.Err
}

// Destroy clears all data and closes the storage engine.
func (g *Engine) Destroy(ctx context.Context) error {
	return g.Err
}

// Ready probes the storage engine and returns an error if it is not ready.
func (g *Engine) Ready(ctx context.Context) error {
	return g.Err
}

// Chore recovers timed out tasks and deletes expired tasks.
func (g *Engine) Chore(ctx context.Context) error {
	return g.Err
}

// Poll makes a promise to claim and execute the next available task in a topic.
func (g *Engine) Poll(ctx context.Context, topic string, p *ratus.Promise) (*ratus.Task, error) {
	return &ratus.Task{
		ID:        p.ID,
		Topic:     topic,
		State:     ratus.TaskStateActive,
		Nonce:     nonce.Generate(ratus.NonceLength),
		Consumer:  p.Consumer,
		Produced:  &cannedDate,
		Scheduled: &cannedDate,
		Consumed:  &cannedDate,
		Deadline:  p.Deadline,
		Payload:   cannedPayload,
	}, g.Err
}

// Commit applies a set of updates to a task and returns the updated task.
func (g *Engine) Commit(ctx context.Context, id string, m *ratus.Commit) (*ratus.Task, error) {
	return &ratus.Task{
		ID:        id,
		Topic:     m.Topic,
		State:     ratus.TaskStateCompleted,
		Produced:  &cannedDate,
		Scheduled: &cannedDate,
		Consumed:  &cannedDate,
		Deadline:  &cannedDate,
		Payload:   cannedPayload,
	}, g.Err
}

// ListTopics lists all topics.
func (g *Engine) ListTopics(ctx context.Context, limit, offset int) ([]*ratus.Topic, error) {
	return []*ratus.Topic{{Name: cannedTopic}}, g.Err
}

// DeleteTopics deletes all topics and tasks.
func (g *Engine) DeleteTopics(ctx context.Context) (*ratus.Deleted, error) {
	return &ratus.Deleted{Deleted: 1}, g.Err
}

// GetTopic gets information about a topic.
func (g *Engine) GetTopic(ctx context.Context, topic string) (*ratus.Topic, error) {
	return &ratus.Topic{Name: cannedTopic, Count: 1}, g.Err
}

// DeleteTopic deletes a topic and its tasks.
func (g *Engine) DeleteTopic(ctx context.Context, topic string) (*ratus.Deleted, error) {
	return &ratus.Deleted{Deleted: 1}, g.Err
}

// ListTasks lists all tasks in a topic.
func (g *Engine) ListTasks(ctx context.Context, topic string, limit, offset int) ([]*ratus.Task, error) {
	return []*ratus.Task{{
		ID:        cannedID,
		Topic:     topic,
		State:     ratus.TaskStatePending,
		Produced:  &cannedDate,
		Scheduled: &cannedDate,
		Consumed:  &cannedDate,
		Deadline:  &cannedDate,
		Payload:   cannedPayload,
	}}, g.Err
}

// InsertTasks inserts a batch of tasks while ignoring existing ones.
func (g *Engine) InsertTasks(ctx context.Context, ts []*ratus.Task) (*ratus.Updated, error) {
	return &ratus.Updated{Created: 1, Updated: 0}, g.Err
}

// UpsertTasks inserts or updates a batch of tasks.
func (g *Engine) UpsertTasks(ctx context.Context, ts []*ratus.Task) (*ratus.Updated, error) {
	return &ratus.Updated{Created: 1, Updated: 1}, g.Err
}

// DeleteTasks deletes all tasks in a topic.
func (g *Engine) DeleteTasks(ctx context.Context, topic string) (*ratus.Deleted, error) {
	return &ratus.Deleted{Deleted: 1}, g.Err
}

// GetTask gets a task by its unique ID.
func (g *Engine) GetTask(ctx context.Context, id string) (*ratus.Task, error) {
	return &ratus.Task{
		ID:        id,
		Topic:     cannedTopic,
		State:     ratus.TaskStatePending,
		Produced:  &cannedDate,
		Scheduled: &cannedDate,
		Consumed:  &cannedDate,
		Deadline:  &cannedDate,
		Payload:   cannedPayload,
	}, g.Err
}

// InsertTask inserts a new task.
func (g *Engine) InsertTask(ctx context.Context, t *ratus.Task) (*ratus.Updated, error) {
	return &ratus.Updated{Created: 1, Updated: 0}, g.Err
}

// UpsertTask inserts or updates a task.
func (g *Engine) UpsertTask(ctx context.Context, t *ratus.Task) (*ratus.Updated, error) {
	return &ratus.Updated{Created: 0, Updated: 1}, g.Err
}

// DeleteTask deletes a task by its unique ID.
func (g *Engine) DeleteTask(ctx context.Context, id string) (*ratus.Deleted, error) {
	return &ratus.Deleted{Deleted: 1}, g.Err
}

// ListPromises lists all promises in a topic.
func (g *Engine) ListPromises(ctx context.Context, topic string, limit, offset int) ([]*ratus.Promise, error) {
	return []*ratus.Promise{{
		ID:       cannedID,
		Deadline: &cannedDate,
	}}, g.Err
}

// DeletePromises deletes all promises in a topic.
func (g *Engine) DeletePromises(ctx context.Context, topic string) (*ratus.Deleted, error) {
	return &ratus.Deleted{Deleted: 1}, g.Err
}

// GetPromise gets a promise by the unique ID of its target task.
func (g *Engine) GetPromise(ctx context.Context, id string) (*ratus.Promise, error) {
	return &ratus.Promise{
		ID:       id,
		Deadline: &cannedDate,
	}, g.Err
}

// InsertPromise makes a promise to claim and execute a task if it is in pending state.
func (g *Engine) InsertPromise(ctx context.Context, p *ratus.Promise) (*ratus.Task, error) {
	return &ratus.Task{
		ID:        cannedID,
		Topic:     cannedTopic,
		State:     ratus.TaskStateActive,
		Nonce:     nonce.Generate(ratus.NonceLength),
		Produced:  &cannedDate,
		Scheduled: &cannedDate,
		Consumed:  &cannedDate,
		Deadline:  &cannedDate,
		Payload:   cannedPayload,
	}, g.Err
}

// UpsertPromise makes a promise to claim and execute a task regardless of its current state.
func (g *Engine) UpsertPromise(ctx context.Context, p *ratus.Promise) (*ratus.Task, error) {
	return &ratus.Task{
		ID:        cannedID,
		Topic:     cannedTopic,
		State:     ratus.TaskStateActive,
		Nonce:     nonce.Generate(ratus.NonceLength),
		Produced:  &cannedDate,
		Scheduled: &cannedDate,
		Consumed:  &cannedDate,
		Deadline:  &cannedDate,
		Payload:   cannedPayload,
	}, g.Err
}

// DeletePromise deletes a promise by the unique ID of its target task.
func (g *Engine) DeletePromise(ctx context.Context, id string) (*ratus.Deleted, error) {
	return &ratus.Deleted{Deleted: 1}, g.Err
}
