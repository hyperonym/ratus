package ratus_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/hyperonym/ratus"
)

func TestError(t *testing.T) {
	t.Run("unmarshal", func(t *testing.T) {
		t.Parallel()
		var r ratus.Error
		r.Error.Code = 42
		r.Error.Message = "hello world"
		if err := r.Err(); err.Error() != r.Error.Message {
			t.Errorf("%q does not match the original error message %q", err.Error(), r.Error.Message)
		}
	})

	t.Run("closed", func(t *testing.T) {
		t.Parallel()
		if e := ratus.NewError(context.Canceled); e.Error.Code != ratus.StatusClientClosedRequest {
			t.Errorf("incorrect error code %d for %q", e.Error.Code, context.Canceled)
		}
		if e := ratus.NewError(io.ErrUnexpectedEOF); e.Error.Code != ratus.StatusClientClosedRequest {
			t.Errorf("incorrect error code %d for %q", e.Error.Code, io.ErrUnexpectedEOF)
		}
	})

	t.Run("sentinel", func(t *testing.T) {
		t.Parallel()
		var s = []error{
			ratus.ErrBadRequest,
			ratus.ErrNotFound,
			ratus.ErrConflict,
			ratus.ErrClientClosedRequest,
			ratus.ErrInternalServerError,
			ratus.ErrServiceUnavailable,
		}
		w := make([]error, len(s))
		for i, err := range s {
			w[i] = fmt.Errorf("%w: %d", err, time.Now().UnixMicro())
		}
		m := make([]*ratus.Error, len(w))
		for i, err := range w {
			m[i] = ratus.NewError(err)
		}
		u := make([]error, len(m))
		for i, e := range m {
			u[i] = e.Err()
		}
		for i, err := range u {
			if !errors.Is(err, s[i]) {
				t.Errorf("%q must be in the error chain of %q", s[i], err)
			}
			if err.Error() != w[i].Error() {
				t.Errorf("%q does not match the original error message %q", err.Error(), w[i].Error())
			}
		}
	})
}
