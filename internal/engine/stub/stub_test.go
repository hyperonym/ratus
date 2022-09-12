package stub_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/engine/stub"
)

func assertExist(t *testing.T, v any) {
	if v == nil {
		t.Fail()
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Error(err)
		return
	}
	if len(b) < 8 {
		t.Fail()
	}
}

func assertError(t *testing.T, err, target error) {
	switch {
	case err == nil && target != nil:
		t.Fail()
	case err != nil && target == nil:
		t.Fail()
	case !errors.Is(err, target):
		t.Fail()
	}
}

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

			t.Run("open", func(t *testing.T) {
				t.Parallel()
				assertError(t, g.Open(ctx), p.err)
			})

			t.Run("close", func(t *testing.T) {
				t.Parallel()
				assertError(t, g.Close(ctx), p.err)
			})

			t.Run("destroy", func(t *testing.T) {
				t.Parallel()
				assertError(t, g.Destroy(ctx), p.err)
			})

			t.Run("ready", func(t *testing.T) {
				t.Parallel()
				assertError(t, g.Ready(ctx), p.err)
			})

			t.Run("chore", func(t *testing.T) {
				t.Parallel()
				assertError(t, g.Chore(ctx), p.err)
			})

			t.Run("poll", func(t *testing.T) {
				t.Parallel()
				v, err := g.Poll(ctx, "id", &ratus.Promise{})
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("commit", func(t *testing.T) {
				t.Parallel()
				v, err := g.Commit(ctx, "id", &ratus.Commit{})
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("list-topics", func(t *testing.T) {
				t.Parallel()
				v, err := g.ListTopics(ctx, 10, 0)
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("delete-topics", func(t *testing.T) {
				t.Parallel()
				v, err := g.DeleteTopics(ctx)
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("get-topic", func(t *testing.T) {
				t.Parallel()
				v, err := g.GetTopic(ctx, "topic")
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("delete-topic", func(t *testing.T) {
				t.Parallel()
				v, err := g.DeleteTopic(ctx, "topic")
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("list-tasks", func(t *testing.T) {
				t.Parallel()
				v, err := g.ListTasks(ctx, "topic", 10, 0)
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("insert-tasks", func(t *testing.T) {
				t.Parallel()
				v, err := g.InsertTasks(ctx, make([]*ratus.Task, 0))
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("upsert-tasks", func(t *testing.T) {
				t.Parallel()
				v, err := g.UpsertTasks(ctx, make([]*ratus.Task, 0))
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("delete-tasks", func(t *testing.T) {
				t.Parallel()
				v, err := g.DeleteTasks(ctx, "topic")
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("get-task", func(t *testing.T) {
				t.Parallel()
				v, err := g.GetTask(ctx, "id")
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("insert-task", func(t *testing.T) {
				t.Parallel()
				v, err := g.InsertTask(ctx, &ratus.Task{})
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("upsert-task", func(t *testing.T) {
				t.Parallel()
				v, err := g.UpsertTask(ctx, &ratus.Task{})
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("delete-task", func(t *testing.T) {
				t.Parallel()
				v, err := g.DeleteTask(ctx, "id")
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("list-promises", func(t *testing.T) {
				t.Parallel()
				v, err := g.ListPromises(ctx, "topic", 10, 0)
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("delete-promises", func(t *testing.T) {
				t.Parallel()
				v, err := g.DeletePromises(ctx, "topic")
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("get-promise", func(t *testing.T) {
				t.Parallel()
				v, err := g.GetPromise(ctx, "id")
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("insert-promise", func(t *testing.T) {
				t.Parallel()
				v, err := g.InsertPromise(ctx, &ratus.Promise{})
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("upsert-promise", func(t *testing.T) {
				t.Parallel()
				v, err := g.UpsertPromise(ctx, &ratus.Promise{})
				assertExist(t, v)
				assertError(t, err, p.err)
			})

			t.Run("delete-promise", func(t *testing.T) {
				t.Parallel()
				v, err := g.DeletePromise(ctx, "id")
				assertExist(t, v)
				assertError(t, err, p.err)
			})
		})
	}
}
