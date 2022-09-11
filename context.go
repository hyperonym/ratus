package ratus

import (
	"context"
	"errors"
	"time"
)

// Context wraps around context.Context to carry scoped values throughout the
// poll-execute-commit workflow. Its deadline will be automatically set based
// on the execution deadline of the acquired task. It also provides chainable
// methods for setting up commits. Since the custom context is only used in
// parameters and return values, it is not considered anti-pattern.
// Reference: https://github.com/golang/go/issues/22602
type Context struct {
	context.Context
	cancel    context.CancelFunc
	committed bool
	commit    Commit
	client    *Client

	// Task acquired by the polling operation.
	Task *Task
}

// Commit applies updates to the acquired task.
func (ctx *Context) Commit() error {

	// Check whether or not a commit has been made to support both automatic
	// acknowledgement and manual commits.
	if ctx.committed {
		return nil
	}

	// The client that called the poll operation will be associated with the
	// returned context. Users should not create new context instances outside
	// of this package.
	if ctx.client == nil {
		return errors.New("cannot commit without an associated client")
	}
	if _, err := ctx.client.PatchTask(ctx.Context, ctx.Task.ID, &ctx.commit); err != nil {
		return err
	}

	// Update committed flag and cancel timeout on success.
	ctx.committed = true
	if ctx.cancel != nil {
		ctx.cancel()
	}

	return nil
}

// SetNonce sets the value for the Nonce field of the commit.
func (ctx *Context) SetNonce(nonce string) *Context {
	ctx.commit.Nonce = nonce
	return ctx
}

// SetTopic sets the value for the Topic field of the commit.
func (ctx *Context) SetTopic(topic string) *Context {
	ctx.commit.Topic = topic
	return ctx
}

// SetState sets the value for the State field of the commit.
func (ctx *Context) SetState(s TaskState) *Context {
	ctx.commit.State = &s
	return ctx
}

// SetScheduled sets the value for the Scheduled field of the commit.
func (ctx *Context) SetScheduled(t time.Time) *Context {
	ctx.commit.Scheduled = &t
	return ctx
}

// SetPayload sets the value for the Payload field of the commit.
func (ctx *Context) SetPayload(v any) *Context {
	ctx.commit.Payload = v
	return ctx
}

// SetDefer sets the value for the Defer field of the commit.
func (ctx *Context) SetDefer(duration string) *Context {
	ctx.commit.Defer = duration
	return ctx
}

// Force sets the Nonce field of the commit to empty to allow force commits.
func (ctx *Context) Force() *Context {
	ctx.commit.Nonce = ""
	return ctx
}

// Abstain is equivalent to calling SetState(TaskStatePending).
func (ctx *Context) Abstain() *Context {
	return ctx.SetState(TaskStatePending)
}

// Archive is equivalent to calling SetState(TaskStateArchived).
func (ctx *Context) Archive() *Context {
	return ctx.SetState(TaskStateArchived)
}

// Reschedule is equivalent to calling Abstain followed by SetScheduled(t).
func (ctx *Context) Reschedule(t time.Time) *Context {
	return ctx.Abstain().SetScheduled(t)
}

// Retry is equivalent to calling Abstain followed by SetDefer(duration).
func (ctx *Context) Retry(duration string) *Context {
	return ctx.Abstain().SetDefer(duration)
}
