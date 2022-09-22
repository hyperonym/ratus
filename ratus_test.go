package ratus_test

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/hyperonym/ratus"
)

func TestEncoding(t *testing.T) {
	n := time.Now()
	m := map[string]any{
		"empty":  nil,
		"bool":   true,
		"int":    123,
		"float":  3.14,
		"string": "hello",
		"array":  []any{1, 2, "a"},
		"nested": map[string]any{
			"empty":  nil,
			"bool":   true,
			"int":    123,
			"float":  3.14,
			"string": "hello",
			"array":  []any{1, 2, "a"},
		},
	}
	ps := []struct {
		name string
		task *ratus.Task
	}{
		{"empty", &ratus.Task{ID: "1", State: ratus.TaskStateActive, Scheduled: &n}},
		{"bool", &ratus.Task{ID: "1", State: ratus.TaskStateActive, Scheduled: &n, Payload: true}},
		{"int", &ratus.Task{ID: "1", State: ratus.TaskStateActive, Scheduled: &n, Payload: 123}},
		{"float", &ratus.Task{ID: "1", State: ratus.TaskStateActive, Scheduled: &n, Payload: 3.14}},
		{"string", &ratus.Task{ID: "1", State: ratus.TaskStateActive, Scheduled: &n, Payload: "hello"}},
		{"array", &ratus.Task{ID: "1", State: ratus.TaskStateActive, Scheduled: &n, Payload: []any{1, 2, "a"}}},
		{"nested", &ratus.Task{ID: "1", State: ratus.TaskStateActive, Scheduled: &n, Payload: m}},
	}

	assert := func(expected, actual *ratus.Task) {
		t.Helper()
		if actual.ID != expected.ID {
			t.Fail()
		}
		if actual.State != expected.State {
			t.Fail()
		}
		if !actual.Scheduled.Round(time.Millisecond * 10).Equal(expected.Scheduled.Round(time.Millisecond * 10)) {
			t.Errorf("incorrect scheduled time, expected %v, got %v", expected.Scheduled, actual.Scheduled)
		}
		e, _ := json.Marshal(expected.Payload)
		a, _ := json.Marshal(actual.Payload)
		if !bytes.Equal(e, a) {
			t.Errorf("incorrect payload, expected %s, got %s", string(e), string(a))
		}
	}

	t.Run("bson", func(t *testing.T) {
		t.Parallel()
		for _, x := range ps {
			p := x
			t.Run(p.name, func(t *testing.T) {
				t.Parallel()
				var b []byte

				t.Run("encode", func(t *testing.T) {
					var err error
					b, err = bson.Marshal(p.task)
					if err != nil {
						t.Error(err)
					}
				})

				t.Run("decode", func(t *testing.T) {
					var v ratus.Task
					if err := bson.Unmarshal(b, &v); err != nil {
						t.Error(err)
					}
					assert(p.task, &v)
				})
			})
		}

		t.Run("error", func(t *testing.T) {
			t.Parallel()
			b, err := bson.Marshal(&struct {
				Topic int `bson:"topic"`
			}{1})
			if err != nil {
				t.Error(err)
			}
			var f ratus.Task
			if err := bson.Unmarshal(b, &f); err == nil {
				t.Fail()
			}
		})
	})

	t.Run("binary", func(t *testing.T) {
		t.Parallel()
		for _, x := range ps {
			p := x
			t.Run(p.name, func(t *testing.T) {
				t.Parallel()
				var buf bytes.Buffer
				enc := gob.NewEncoder(&buf)
				dec := gob.NewDecoder(&buf)

				t.Run("encode", func(t *testing.T) {
					if err := enc.Encode(p.task); err != nil {
						t.Error(err)
					}
				})

				t.Run("decode", func(t *testing.T) {
					var v ratus.Task
					if err := dec.Decode(&v); err != nil {
						t.Error(err)
					}
					assert(p.task, &v)
				})
			})
		}
	})
}

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
