// Package ratus contains data models and a client library for Go applications.
package ratus

// TaskState indicates the state of a task.
type TaskState int32

const (
	// The "pending" state indicates that the task is ready to be executed or
	// is waiting to be executed in the future.
	TaskStatePending TaskState = iota

	// The "active" state indicates that the task is being processed by a
	// consumer. Active tasks that have timed out will be automatically reset
	// to the "pending" state. Consumer code should handle failure and set the
	// state to "pending" to retry later if necessary.
	TaskStateActive

	// The "completed" state indicates that the task has completed its execution.
	// If the storage engine implementation supports TTL, completed tasks will
	// be automatically deleted after the retention period has expired.
	TaskStateCompleted

	// The "archived" state indicates that the task is stored as an archive.
	// Archived tasks will never be deleted due to expiration.
	TaskStateArchived
)
