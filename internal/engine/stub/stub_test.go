package stub_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/engine/stub"
)

func TestStub(t *testing.T) {
	for _, x := range []struct {
		name string
		err  error
	}{
		{"normal", nil},
		{"conflict", ratus.ErrConflict},
	} {
		p := x
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()
			g := stub.Engine{p.err}
			ctx := context.Background()

			for _, x := range []struct {
				name string
				test func() error
			}{
				{"open", func() error { return g.Open(ctx) }},
				{"close", func() error { return g.Close(ctx) }},
				{"destroy", func() error { return g.Destroy(ctx) }},
				{"ready", func() error { return g.Ready(ctx) }},
				{"chore", func() error { return g.Chore(ctx) }},
			} {
				q := x
				t.Run(q.name, func(t *testing.T) {
					t.Parallel()
					if !errors.Is(q.test(), p.err) {
						t.Fail()
					}
				})
			}

			for _, x := range []struct {
				name string
				test func() (any, error)
			}{
				{"poll", func() (any, error) { return g.Poll(ctx, "id", &ratus.Promise{}) }},
				{"commit", func() (any, error) { return g.Commit(ctx, "id", &ratus.Commit{}) }},
				{"list-topics", func() (any, error) { return g.ListTopics(ctx, 10, 0) }},
				{"delete-topics", func() (any, error) { return g.DeleteTopics(ctx) }},
				{"get-topic", func() (any, error) { return g.GetTopic(ctx, "topic") }},
				{"delete-topic", func() (any, error) { return g.DeleteTopic(ctx, "topic") }},
				{"list-tasks", func() (any, error) { return g.ListTasks(ctx, "topic", 10, 0) }},
				{"insert-tasks", func() (any, error) { return g.InsertTasks(ctx, make([]*ratus.Task, 0)) }},
				{"upsert-tasks", func() (any, error) { return g.UpsertTasks(ctx, make([]*ratus.Task, 0)) }},
				{"delete-tasks", func() (any, error) { return g.DeleteTasks(ctx, "topic") }},
				{"get-task", func() (any, error) { return g.GetTask(ctx, "id") }},
				{"insert-task", func() (any, error) { return g.InsertTask(ctx, &ratus.Task{}) }},
				{"upsert-task", func() (any, error) { return g.UpsertTask(ctx, &ratus.Task{}) }},
				{"delete-task", func() (any, error) { return g.DeleteTask(ctx, "id") }},
				{"list-promises", func() (any, error) { return g.ListPromises(ctx, "topic", 10, 0) }},
				{"delete-promises", func() (any, error) { return g.DeletePromises(ctx, "topic") }},
				{"get-promise", func() (any, error) { return g.GetPromise(ctx, "id") }},
				{"insert-promise", func() (any, error) { return g.InsertPromise(ctx, &ratus.Promise{}) }},
				{"upsert-promise", func() (any, error) { return g.UpsertPromise(ctx, &ratus.Promise{}) }},
				{"delete-promise", func() (any, error) { return g.DeletePromise(ctx, "id") }},
			} {
				q := x
				t.Run(q.name, func(t *testing.T) {
					t.Parallel()
					v, err := q.test()
					if v == nil {
						t.Fail()
					}
					if !errors.Is(err, p.err) {
						t.Fail()
					}
				})
			}
		})
	}
}
