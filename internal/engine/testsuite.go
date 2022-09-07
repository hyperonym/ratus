package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/hyperonym/ratus"
)

// Test runs a collection of test cases that are grouped for testing storage
// engine implementations against the provided engine instance.
func Test(t *testing.T, g Engine) {
	ctx := context.Background()

	// Test life cycle.
	if err := g.Open(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := g.Destroy(ctx); err != nil {
			t.Error(err)
		}
	})
	if err := g.Ready(ctx); err != nil {
		t.Fatal(err)
	}

	// Test operations in the blank state.
	t.Run("blank", func(t *testing.T) {
		t.Run("chore", func(t *testing.T) {
			t.Parallel()
			if err := g.Chore(ctx); err != nil {
				t.Error()
			}
		})

		t.Run("poll", func(t *testing.T) {
			t.Parallel()
			if _, err := g.Poll(ctx, "test", &ratus.Promise{ID: "foo"}); err == nil || !errors.Is(err, ratus.ErrNotFound) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrNotFound, err)
			}
		})

		t.Run("commit", func(t *testing.T) {
			t.Parallel()
			if _, err := g.Commit(ctx, "foo", &ratus.Commit{Payload: "hello"}); err == nil || !errors.Is(err, ratus.ErrNotFound) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrNotFound, err)
			}
		})

		t.Run("topic", func(t *testing.T) {
			t.Parallel()
			v, err := g.ListTopics(ctx, 10, 0)
			if err != nil {
				t.Error(err)
			}
			if len(v) != 0 {
				t.Errorf("incorrect number of results, expected 0, got %d", len(v))
			}
			if _, err := g.GetTopic(ctx, "test"); err == nil || !errors.Is(err, ratus.ErrNotFound) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrNotFound, err)
			}
			d, err := g.DeleteTopic(ctx, "test")
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 0 {
				t.Errorf("incorrect number of deletions, expected 0, got %d", d.Deleted)
			}
			d, err = g.DeleteTopics(ctx)
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 0 {
				t.Errorf("incorrect number of deletions, expected 0, got %d", d.Deleted)
			}
		})

		t.Run("task", func(t *testing.T) {
			t.Parallel()
			v, err := g.ListTasks(ctx, "test", 10, 0)
			if err != nil {
				t.Error(err)
			}
			if len(v) != 0 {
				t.Errorf("incorrect number of results, expected 0, got %d", len(v))
			}
			if _, err := g.GetTask(ctx, "foo"); err == nil || !errors.Is(err, ratus.ErrNotFound) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrNotFound, err)
			}
			d, err := g.DeleteTask(ctx, "foo")
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 0 {
				t.Errorf("incorrect number of deletions, expected 0, got %d", d.Deleted)
			}
			d, err = g.DeleteTasks(ctx, "test")
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 0 {
				t.Errorf("incorrect number of deletions, expected 0, got %d", d.Deleted)
			}
		})

		t.Run("promise", func(t *testing.T) {
			t.Parallel()
			v, err := g.ListPromises(ctx, "test", 10, 0)
			if err != nil {
				t.Error(err)
			}
			if len(v) != 0 {
				t.Errorf("incorrect number of results, expected 0, got %d", len(v))
			}
			if _, err := g.GetPromise(ctx, "foo"); err == nil || !errors.Is(err, ratus.ErrNotFound) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrNotFound, err)
			}
			d, err := g.DeletePromise(ctx, "foo")
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 0 {
				t.Errorf("incorrect number of deletions, expected 0, got %d", d.Deleted)
			}
			d, err = g.DeletePromises(ctx, "test")
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 0 {
				t.Errorf("incorrect number of deletions, expected 0, got %d", d.Deleted)
			}
		})
	})
}
