// Package ratus contains data models and a client library for Go applications.
package ratus

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultTimeout is the default timeout duration for task execution.
const DefaultTimeout = "10m"

// DefaultLimit is the default number of resources to return in pagination.
const DefaultLimit = 10

// NonceLength is the length of the randomly generated nonce strings.
const NonceLength = 16

// StatusClientClosedRequest is the code for client closed request errors.
const StatusClientClosedRequest = 499

var (
	// ErrBadRequest is returned when the request is malformed.
	ErrBadRequest = errors.New("bad request")

	// ErrNotFound is returned when the requested resource is not found.
	ErrNotFound = errors.New("not found")

	// ErrConflict is returned when the resource conflicts with existing ones.
	ErrConflict = errors.New("conflict")

	// ErrClientClosedRequest is returned when the client closed the request.
	ErrClientClosedRequest = errors.New("client closed request")

	// ErrInternalServerError is returned when the server encountered a
	// situation it does not know how to handle.
	ErrInternalServerError = errors.New("internal server error")

	// ErrServiceUnavailable is returned when the service is unavailable.
	ErrServiceUnavailable = errors.New("service unavailable")
)

// init registers interface types for binary encoding and decoding.
func init() {
	gob.Register([]interface{}{})
	gob.Register(map[string]interface{}{})
	gob.Register([]map[string]interface{}{})
}

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

// Topic refers to an ordered subset of tasks with the same topic name property.
type Topic struct {

	// User-defined unique name of the topic.
	Name string `json:"name" bson:"_id"`

	// The number of tasks that belong to the topic.
	Count int64 `json:"count,omitempty" bson:"count,omitempty"`
}

// Task references an idempotent unit of work that should be executed asynchronously.
type Task struct {

	// User-defined unique ID of the task.
	// Task IDs across all topics share the same namespace.
	ID string `json:"_id" bson:"_id"`

	// Topic that the task currently belongs to. Tasks under the same topic
	// will be executed according to the scheduled time.
	Topic string `json:"topic" bson:"topic"`

	// Current state of the task. At a given moment, the state of a task may be
	// either "pending", "active", "completed" or "archived".
	State TaskState `json:"state" bson:"state"`

	// The nonce field stores a random string for implementing an optimistic
	// concurrency control (OCC) layer outside of the storage engine. Ratus
	// ensures consumers can only commit to tasks that have not changed since
	// the promise was made by verifying the nonce field.
	Nonce string `json:"nonce" bson:"nonce"`

	// Identifier of the producer instance who produced the task.
	Producer string `json:"producer,omitempty" bson:"producer,omitempty"`
	// Identifier of the consumer instance who consumed the task.
	Consumer string `json:"consumer,omitempty" bson:"consumer,omitempty"`

	// The time the task was created.
	// Timestamps are generated by the instance running Ratus, remember to
	// perform clock synchronization before running multiple instances.
	Produced *time.Time `json:"produced,omitempty" bson:"produced,omitempty"`
	// The time the task is scheduled to be executed. Tasks will not be
	// executed until the scheduled time arrives. After the scheduled time,
	// excessive tasks will be executed in the order of the scheduled time.
	Scheduled *time.Time `json:"scheduled,omitempty" bson:"scheduled,omitempty"`
	// The time the task was claimed by a consumer.
	// Not to confuse this with the time of commit, which is not recorded.
	Consumed *time.Time `json:"consumed,omitempty" bson:"consumed,omitempty"`
	// The deadline for the completion of execution promised by the consumer.
	// Consumer code needs to commit the task before this deadline, otherwise
	// the task is determined to have timed out and will be reset to the
	// "pending" state, allowing other consumers to retry.
	Deadline *time.Time `json:"deadline,omitempty" bson:"deadline,omitempty"`

	// A minimal descriptor of the task to be executed.
	// It is not recommended to rely on Ratus as the main storage of tasks.
	// Instead, consider storing the complete task record in a database, and
	// use a minimal descriptor as the payload to reference the task.
	Payload any `json:"payload,omitempty" bson:"payload,omitempty"`

	// A duration relative to the time the task is accepted, indicating that
	// the task will be scheduled to execute after this duration. When the
	// absolute scheduled time is specified, the scheduled time will take
	// precedence. It is recommended to use relative durations whenever
	// possible to avoid clock synchronization issues. The value must be a
	// valid duration string parsable by time.ParseDuration. This field is only
	// used when creating a task and will be cleared after converting to an
	// absolute scheduled time.
	Defer string `json:"defer,omitempty" bson:"-"`
}

// Decode parses the payload of the task and stores the result in the value
// pointed by the specified pointer.
func (t *Task) Decode(v any) error {

	// Counterintuitively, the seemingly dumb approach of just marshalling
	// input into JSON bytes and decoding it from those bytes is actually both
	// 29.5% faster (than reflection) and causes less memory allocations.
	// Reference: https://github.com/mitchellh/mapstructure/issues/37
	b, err := json.Marshal(t.Payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// Promise represents a claim on the ownership of an active task.
type Promise struct {

	// Unique ID of the promise, which is the same as the target task ID.
	// A promise with an empty ID is considered an "wildcard promise", and
	// Ratus will assign an appropriate task based on the status of the queue.
	// A task can only be owned by a single promise at a given time.
	ID string `json:"_id,omitempty" bson:"_id" form:"_id"`

	// Identifier of the consumer instance who consumed the task.
	Consumer string `json:"consumer,omitempty" bson:"consumer,omitempty" form:"consumer"`

	// The deadline for the completion of execution promised by the consumer.
	// Consumer code needs to commit the task before this deadline, otherwise
	// the task is determined to have timed out and will be reset to the
	// "pending" state, allowing other consumers to retry.
	Deadline *time.Time `json:"deadline,omitempty" bson:"deadline,omitempty" form:"deadline"`

	// Timeout duration for task execution promised by the consumer. When the
	// absolute deadline time is specified, the deadline will take precedence.
	// It is recommended to use relative durations whenever possible to avoid
	// clock synchronization issues. The value must be a valid duration string
	// parsable by time.ParseDuration. This field is only used when creating a
	// promise and will be cleared after converting to an absolute deadline.
	Timeout string `json:"timeout,omitempty" bson:"-" form:"timeout"`
}

// Commit contains a set of updates to be applied to a task.
type Commit struct {

	// If not empty, the commit will be accepted only if the value matches the
	// corresponding nonce of the target task.
	Nonce string `json:"nonce,omitempty" bson:"nonce,omitempty"`

	// If not empty, transfer the task to the specified topic.
	Topic string `json:"topic,omitempty" bson:"topic,omitempty"`

	// If not nil, set the state of the task to the specified value.
	// If nil, the state of the task will be set to "completed" by default.
	State *TaskState `json:"state,omitempty" bson:"state,omitempty"`

	// If not nil, set the scheduled time of the task to the specified value.
	Scheduled *time.Time `json:"scheduled,omitempty" bson:"scheduled,omitempty"`

	// If not nil, use this value to replace the payload of the task.
	Payload any `json:"payload,omitempty" bson:"payload,omitempty"`

	// A duration relative to the time the commit is accepted, indicating that
	// the task will be scheduled to execute after this duration. When the
	// absolute scheduled time is specified, the scheduled time will take
	// precedence. It is recommended to use relative durations whenever
	// possible to avoid clock synchronization issues. The value must be a
	// valid duration string parsable by time.ParseDuration. This field is only
	// used when creating a commit and will be cleared after converting to an
	// absolute scheduled time.
	Defer string `json:"defer,omitempty" bson:"-"`
}

// Topics contains a list of topic resources.
type Topics struct {
	Data []*Topic `json:"data"`
}

// Tasks contains a list of task resources.
type Tasks struct {
	Data []*Task `json:"data"`
}

// Promises contains a list of promise resources.
type Promises struct {
	Data []*Promise `json:"data"`
}

// Updated contains result of an update operation.
type Updated struct {

	// Number of resources created by the operation.
	Created int64 `json:"created"`

	// Number of resources updated by the operation.
	Updated int64 `json:"updated"`
}

// Deleted contains result of a delete operation.
type Deleted struct {

	// Number of resources deleted by the operation.
	Deleted int64 `json:"deleted"`
}

// Error contains an error message.
type Error struct {

	// The error object.
	Error struct {

		// Code of the error.
		Code int `json:"code"`

		// Message of the error.
		Message string `json:"message"`
	} `json:"error"`
}

// Err returns an error type from the error message. It will automatically wrap
// sentinel error types based on the code and remove duplicates in the message.
func (e *Error) Err() error {

	// Determine the sentinel error to wrap around based on the error code.
	var err error
	switch e.Error.Code {
	case StatusClientClosedRequest:
		err = ErrClientClosedRequest
	case http.StatusBadRequest:
		err = ErrBadRequest
	case http.StatusNotFound:
		err = ErrNotFound
	case http.StatusConflict:
		err = ErrConflict
	case http.StatusInternalServerError:
		err = ErrInternalServerError
	case http.StatusServiceUnavailable:
		err = ErrServiceUnavailable
	default:
		return errors.New(e.Error.Message)
	}

	// Remove duplicated prefix caused by wrapping.
	m := strings.TrimPrefix(e.Error.Message, err.Error())
	if m != "" {
		err = fmt.Errorf("%w%s", err, m)
	}

	return err
}

// NewError creates an error message from an error type.
func NewError(err error) *Error {

	// Determine status code for the error.
	var s int
	switch {
	case errors.Is(err, context.Canceled):
		s = StatusClientClosedRequest
	case errors.Is(err, io.ErrUnexpectedEOF):
		s = StatusClientClosedRequest
	case errors.Is(err, ErrClientClosedRequest):
		s = StatusClientClosedRequest
	case errors.Is(err, ErrBadRequest):
		s = http.StatusBadRequest
	case errors.Is(err, ErrNotFound):
		s = http.StatusNotFound
	case errors.Is(err, ErrConflict):
		s = http.StatusConflict
	case errors.Is(err, ErrServiceUnavailable):
		s = http.StatusServiceUnavailable
	default:
		s = http.StatusInternalServerError
	}

	// Populate error information.
	var e Error
	e.Error.Code = s
	e.Error.Message = err.Error()

	return &e
}
