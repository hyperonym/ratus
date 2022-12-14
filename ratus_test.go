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

	"github.com/hyperonym/ratus"
)

func TestTask(t *testing.T) {
	t.Run("encode", func(t *testing.T) {
		t.Parallel()
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
		for _, x := range []struct {
			name string
			task *ratus.Task
		}{
			{"empty", &ratus.Task{ID: "1", Topic: "test", State: ratus.TaskStateActive, Scheduled: &n}},
			{"bool", &ratus.Task{ID: "2", Topic: "test", State: ratus.TaskStateActive, Scheduled: &n, Payload: true}},
			{"int", &ratus.Task{ID: "3", Topic: "test", State: ratus.TaskStateActive, Scheduled: &n, Payload: 123}},
			{"float", &ratus.Task{ID: "4", Topic: "test", State: ratus.TaskStateActive, Scheduled: &n, Payload: 3.14}},
			{"string", &ratus.Task{ID: "5", Topic: "test", State: ratus.TaskStateActive, Scheduled: &n, Payload: "hello"}},
			{"array", &ratus.Task{ID: "6", Topic: "test", State: ratus.TaskStateActive, Scheduled: &n, Payload: []any{1, 2, "a"}}},
			{"nested", &ratus.Task{ID: "7", Topic: "test", State: ratus.TaskStateActive, Scheduled: &n, Payload: m}},
		} {
			p := x
			t.Run(p.name, func(t *testing.T) {
				t.Parallel()
				var buf bytes.Buffer
				enc := gob.NewEncoder(&buf)
				dec := gob.NewDecoder(&buf)
				if err := enc.Encode(p.task); err != nil {
					t.Error(err)
				}
				var v ratus.Task
				if err := dec.Decode(&v); err != nil {
					t.Error(err)
				}
				if v.ID != p.task.ID {
					t.Fail()
				}
				if v.State != p.task.State {
					t.Fail()
				}
				if !v.Scheduled.Round(time.Millisecond * 10).Equal(p.task.Scheduled.Round(time.Millisecond * 10)) {
					t.Errorf("incorrect scheduled time, expected %v, got %v", p.task.Scheduled, v.Scheduled)
				}
				e, _ := json.Marshal(p.task.Payload)
				a, _ := json.Marshal(v.Payload)
				if !bytes.Equal(e, a) {
					t.Errorf("incorrect payload, expected %s, got %s", string(e), string(a))
				}
			})
		}
	})

	t.Run("decode", func(t *testing.T) {
		t.Parallel()

		t.Run("empty", func(t *testing.T) {
			t.Parallel()
			var p any
			v := ratus.Task{Payload: nil}
			if err := v.Decode(&p); err != nil {
				t.Error(err)
			}
			if p != nil {
				t.Fail()
			}
		})

		t.Run("bool", func(t *testing.T) {
			t.Parallel()
			var p bool
			v := ratus.Task{Payload: true}
			if err := v.Decode(&p); err != nil {
				t.Error(err)
			}
			if !p {
				t.Fail()
			}
		})

		t.Run("int", func(t *testing.T) {
			t.Parallel()
			var p int
			v := ratus.Task{Payload: 123}
			if err := v.Decode(&p); err != nil {
				t.Error(err)
			}
			if p != 123 {
				t.Fail()
			}
		})

		t.Run("float", func(t *testing.T) {
			t.Parallel()
			var p float32
			v := ratus.Task{Payload: 3.14}
			if err := v.Decode(&p); err != nil {
				t.Error(err)
			}
			if p != 3.14 {
				t.Fail()
			}
		})

		t.Run("string", func(t *testing.T) {
			t.Parallel()
			var p string
			v := ratus.Task{Payload: "hello"}
			if err := v.Decode(&p); err != nil {
				t.Error(err)
			}
			if p != "hello" {
				t.Fail()
			}
		})

		t.Run("array", func(t *testing.T) {
			t.Parallel()
			var p []any
			v := ratus.Task{Payload: []any{1, 2, "a"}}
			if err := v.Decode(&p); err != nil {
				t.Error(err)
			}
			if len(p) != 3 {
				t.Fail()
			}
		})

		t.Run("nested", func(t *testing.T) {
			t.Parallel()
			var p struct {
				Name string
				Date struct {
					Month int
					Day   int
				}
			}
			v := ratus.Task{Payload: map[string]any{
				"name": "peak",
				"date": map[string]int{
					"month": 7,
					"day":   29,
				},
			}}
			if err := v.Decode(&p); err != nil {
				t.Error(err)
			}
			if p.Name != "peak" || p.Date.Month != 7 || p.Date.Day != 29 {
				t.Fail()
			}
		})

		t.Run("invalid", func(t *testing.T) {
			t.Parallel()
			var p func()
			v := ratus.Task{Payload: func() {}}
			if err := v.Decode(&p); err == nil {
				t.Fail()
			}
		})
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
