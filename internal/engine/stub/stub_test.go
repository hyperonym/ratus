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
			for _, f := range []func() (any, error){
				func() (any, error) { return nil, g.Open(ctx) },
				func() (any, error) { return nil, g.Close(ctx) },
				func() (any, error) { return nil, g.Destroy(ctx) },
				func() (any, error) { return nil, g.Ready(ctx) },
				func() (any, error) { return nil, g.Chore(ctx) },
				func() (any, error) { return g.Poll(ctx, "id", &ratus.Promise{}) },
				func() (any, error) { return g.Commit(ctx, "id", &ratus.Commit{}) },
				func() (any, error) { return g.ListTopics(ctx, 10, 0) },
				func() (any, error) { return g.DeleteTopics(ctx) },
				func() (any, error) { return g.GetTopic(ctx, "topic") },
				func() (any, error) { return g.DeleteTopic(ctx, "topic") },
				func() (any, error) { return g.ListTasks(ctx, "topic", 10, 0) },
				func() (any, error) { return g.InsertTasks(ctx, make([]*ratus.Task, 0)) },
				func() (any, error) { return g.UpsertTasks(ctx, make([]*ratus.Task, 0)) },
				func() (any, error) { return g.DeleteTasks(ctx, "topic") },
				func() (any, error) { return g.GetTask(ctx, "id") },
				func() (any, error) { return g.InsertTask(ctx, &ratus.Task{}) },
				func() (any, error) { return g.UpsertTask(ctx, &ratus.Task{}) },
				func() (any, error) { return g.DeleteTask(ctx, "id") },
				func() (any, error) { return g.ListPromises(ctx, "topic", 10, 0) },
				func() (any, error) { return g.DeletePromises(ctx, "topic") },
				func() (any, error) { return g.GetPromise(ctx, "id") },
				func() (any, error) { return g.InsertPromise(ctx, &ratus.Promise{}) },
				func() (any, error) { return g.UpsertPromise(ctx, &ratus.Promise{}) },
				func() (any, error) { return g.DeletePromise(ctx, "id") },
			} {
				if _, err := f(); !errors.Is(err, p.err) {
					t.Fail()
				}
			}
		})
	}
}
