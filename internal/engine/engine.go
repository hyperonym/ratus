// Package engine defines the interface for storage engine implementations.
package engine

import (
	"context"

	"github.com/hyperonym/ratus"
)

// Engine defines the interface for storage engine implementations.
type Engine interface {

	// Open or connect to the storage engine.
	Open(ctx context.Context) error
	// Close or disconnect from the storage engine.
	Close(ctx context.Context) error
	// Destroy clears all data and closes the storage engine.
	Destroy(ctx context.Context) error
	// Ready probes the storage engine and returns an error if it is not ready.
	Ready(ctx context.Context) error

	// Chore recovers timed out tasks and deletes expired tasks.
	Chore(ctx context.Context) error
	// Poll claims the next available task in the topic based on the scheduled time.
	Poll(ctx context.Context, topic string, p *ratus.Promise) (*ratus.Task, error)
	// Commit applies a set of updates to a task and returns the updated task.
	Commit(ctx context.Context, id string, m *ratus.Commit) (*ratus.Task, error)

	// ListTopics lists all topics.
	ListTopics(ctx context.Context, limit, offset int) ([]*ratus.Topic, error)
	// DeleteTopics deletes all topics and tasks.
	DeleteTopics(ctx context.Context) (*ratus.Deleted, error)
	// GetTopic gets information about a topic.
	GetTopic(ctx context.Context, topic string) (*ratus.Topic, error)
	// DeleteTopic deletes a topic and its tasks.
	DeleteTopic(ctx context.Context, topic string) (*ratus.Deleted, error)

	// ListTasks lists all tasks in a topic.
	ListTasks(ctx context.Context, topic string, limit, offset int) ([]*ratus.Task, error)
	// InsertTasks inserts a batch of tasks while ignoring existing ones.
	InsertTasks(ctx context.Context, ts []*ratus.Task) (*ratus.Updated, error)
	// UpsertTasks inserts or updates a batch of tasks.
	UpsertTasks(ctx context.Context, ts []*ratus.Task) (*ratus.Updated, error)
	// DeleteTasks deletes all tasks in a topic.
	DeleteTasks(ctx context.Context, topic string) (*ratus.Deleted, error)
	// GetTask gets a task by its unique ID.
	GetTask(ctx context.Context, id string) (*ratus.Task, error)
	// InsertTask inserts a new task.
	InsertTask(ctx context.Context, t *ratus.Task) (*ratus.Updated, error)
	// UpsertTask inserts or updates a task.
	UpsertTask(ctx context.Context, t *ratus.Task) (*ratus.Updated, error)
	// DeleteTask deletes a task by its unique ID.
	DeleteTask(ctx context.Context, id string) (*ratus.Deleted, error)

	// ListPromises lists all promises in a topic.
	ListPromises(ctx context.Context, topic string, limit, offset int) ([]*ratus.Promise, error)
	// DeletePromises deletes all promises in a topic.
	DeletePromises(ctx context.Context, topic string) (*ratus.Deleted, error)
	// GetPromise gets a promise by the unique ID of its target task.
	GetPromise(ctx context.Context, id string) (*ratus.Promise, error)
	// InsertPromise claims the target task if it is in pending state.
	InsertPromise(ctx context.Context, p *ratus.Promise) (*ratus.Task, error)
	// UpsertPromise claims the target task regardless of its current state.
	UpsertPromise(ctx context.Context, p *ratus.Promise) (*ratus.Task, error)
	// DeletePromise deletes a promise by the unique ID of its target task.
	DeletePromise(ctx context.Context, id string) (*ratus.Deleted, error)
}
