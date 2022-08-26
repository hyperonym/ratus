// Package ratus contains data models and a client library for Go applications.
package ratus

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

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
	case 499:
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
		s = 499
	case errors.Is(err, io.ErrUnexpectedEOF):
		s = 499
	case errors.Is(err, ErrClientClosedRequest):
		s = 499
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
