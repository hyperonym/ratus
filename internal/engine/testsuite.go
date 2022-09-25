package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/nonce"
)

// Test runs a collection of test cases that are grouped for testing storage
// engine implementations. The test suite handles the initialization of the
// provided engine instance, and clears all data when the test is completed.
// This test suite only covers the core functionality of a storage engine
// implementation. Test cases for optional features like TTL should be included
// in the package of the specific engine.
func Test(t *testing.T, g Engine) {
	ctx := context.Background()
	if err := g.Ready(ctx); !errors.Is(err, ratus.ErrServiceUnavailable) {
		t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrServiceUnavailable, err)
	}
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
			if _, err := g.Poll(ctx, "test", &ratus.Promise{ID: "foo"}); !errors.Is(err, ratus.ErrNotFound) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrNotFound, err)
			}
		})

		t.Run("commit", func(t *testing.T) {
			t.Parallel()
			if _, err := g.Commit(ctx, "foo", &ratus.Commit{Payload: "hello"}); !errors.Is(err, ratus.ErrNotFound) {
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
			if _, err := g.GetTopic(ctx, "test"); !errors.Is(err, ratus.ErrNotFound) {
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
			if _, err := g.GetTask(ctx, "foo"); !errors.Is(err, ratus.ErrNotFound) {
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
			if _, err := g.GetPromise(ctx, "foo"); !errors.Is(err, ratus.ErrNotFound) {
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

	// Test operations in sequential order.
	t.Run("sequential", func(t *testing.T) {
		n := time.Now()

		t.Run("task", func(t *testing.T) {
			u, err := g.InsertTask(ctx, &ratus.Task{
				ID:        "1",
				Topic:     "test",
				State:     ratus.TaskStatePending,
				Produced:  &n,
				Scheduled: &n,
				Payload:   "a",
			})
			if err != nil {
				t.Error(err)
			}
			if u.Created != 1 {
				t.Errorf("incorrect number of creations, expected 1, got %d", u.Created)
			}
			u, err = g.InsertTasks(ctx, []*ratus.Task{{
				ID:        "1",
				Topic:     "test",
				State:     ratus.TaskStatePending,
				Produced:  &n,
				Scheduled: &n,
				Payload:   "xxx",
			}, {
				ID:        "2",
				Topic:     "test",
				State:     ratus.TaskStatePending,
				Produced:  &n,
				Scheduled: &n,
				Payload:   "b",
			}})
			if err != nil {
				t.Error(err)
			}
			if u.Created != 1 {
				t.Errorf("incorrect number of creations, expected 1, got %d", u.Created)
			}
			if _, err = g.InsertTask(ctx, &ratus.Task{
				ID:        "1",
				Topic:     "test",
				State:     ratus.TaskStatePending,
				Produced:  &n,
				Scheduled: &n,
				Payload:   "xxx",
			}); !errors.Is(err, ratus.ErrConflict) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrConflict, err)
			}
			v, err := g.GetTask(ctx, "1")
			if err != nil {
				t.Error(err)
			}
			if fmt.Sprint(v.Payload) != "a" {
				t.Errorf("incorrect payload in task, expected %q, got %q", "a", v.Payload)
			}
			c, err := g.GetTopic(ctx, "test")
			if err != nil {
				t.Error(err)
			}
			if c.Count != 2 {
				t.Errorf("incorrect number of results, expected 2, got %d", c.Count)
			}
		})

		t.Run("promise", func(t *testing.T) {
			v, err := g.InsertPromise(ctx, &ratus.Promise{
				ID:       "1",
				Deadline: &n,
			})
			if err != nil {
				t.Error(err)
			}
			if v.State != ratus.TaskStateActive {
				t.Errorf("incorrect task state, expected %d, got %d", ratus.TaskStateActive, v.State)
			}
			if _, err := g.InsertPromise(ctx, &ratus.Promise{
				ID:       "1",
				Deadline: &n,
			}); !errors.Is(err, ratus.ErrConflict) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrConflict, err)
			}
			if _, err := g.InsertPromise(ctx, &ratus.Promise{
				ID:       "xxx",
				Deadline: &n,
			}); !errors.Is(err, ratus.ErrNotFound) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrNotFound, err)
			}
			p, err := g.GetPromise(ctx, "1")
			if err != nil {
				t.Error(err)
			}
			if n.Unix() != p.Deadline.Unix() {
				t.Errorf("incorrect promise deadline, expected %v, got %v", n.Unix(), p.Deadline.Unix())
			}
			v, err = g.UpsertPromise(ctx, &ratus.Promise{
				ID:       "2",
				Deadline: &n,
			})
			if err != nil {
				t.Error(err)
			}
			if v.State != ratus.TaskStateActive {
				t.Errorf("incorrect task state, expected %d, got %d", ratus.TaskStateActive, v.State)
			}
			if _, err := g.UpsertPromise(ctx, &ratus.Promise{
				ID:       "xxx",
				Deadline: &n,
			}); !errors.Is(err, ratus.ErrNotFound) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrNotFound, err)
			}
			p, err = g.GetPromise(ctx, "2")
			if err != nil {
				t.Error(err)
			}
			if n.Unix() != p.Deadline.Unix() {
				t.Errorf("incorrect promise deadline, expected %v, got %v", n.Unix(), p.Deadline.Unix())
			}
		})

		t.Run("chore", func(t *testing.T) {
			if err := g.Chore(ctx); err != nil {
				t.Error(err)
			}
			if _, err := g.GetPromise(ctx, "1"); !errors.Is(err, ratus.ErrNotFound) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrNotFound, err)
			}
		})

		t.Run("poll", func(t *testing.T) {
			if _, err := g.Poll(ctx, "test", &ratus.Promise{Deadline: &n}); err != nil {
				t.Error(err)
			}
			if _, err := g.Poll(ctx, "test", &ratus.Promise{Deadline: &n}); err != nil {
				t.Error(err)
			}
			v, err := g.ListPromises(ctx, "test", 10, 0)
			if err != nil {
				t.Error(err)
			}
			if len(v) != 2 {
				t.Errorf("incorrect number of results, expected 2, got %d", len(v))
			}
		})

		t.Run("commit", func(t *testing.T) {
			if _, err := g.Commit(ctx, "1", &ratus.Commit{Nonce: "xxx"}); !errors.Is(err, ratus.ErrConflict) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrConflict, err)
			}
			if _, err := g.Commit(ctx, "xxx", &ratus.Commit{Nonce: "xxx"}); !errors.Is(err, ratus.ErrNotFound) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrNotFound, err)
			}
			v, err := g.GetTask(ctx, "1")
			if err != nil {
				t.Error(err)
			}
			s := ratus.TaskStateCompleted
			m := &ratus.Commit{
				Nonce:     v.Nonce,
				Topic:     "completed",
				State:     &s,
				Scheduled: &n,
				Payload:   "completed",
			}
			v, err = g.Commit(ctx, "1", m)
			if err != nil {
				t.Error(err)
			}
			if fmt.Sprint(v.Payload) != "completed" {
				t.Errorf("incorrect payload in task, expected %q, got %q", "completed", v.Payload)
			}
			if _, err := g.Commit(ctx, "1", m); err == nil {
				t.Error("failed to invalidate duplicated commits")
			}
		})

		t.Run("topic", func(t *testing.T) {
			v, err := g.ListTopics(ctx, 10, 0)
			if err != nil {
				t.Error(err)
			}
			if len(v) != 2 {
				t.Errorf("incorrect number of results, expected 2, got %d", len(v))
			}
			d, err := g.DeleteTopic(ctx, "completed")
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 1 {
				t.Errorf("incorrect number of deletions, expected 1, got %d", d.Deleted)
			}
			d, err = g.DeleteTopics(ctx)
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 1 {
				t.Errorf("incorrect number of deletions, expected 1, got %d", d.Deleted)
			}
		})

		t.Run("clean", func(t *testing.T) {
			d, err := g.DeleteTopics(ctx)
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 0 {
				t.Errorf("incorrect number of deletions, expected 0, got %d", d.Deleted)
			}
		})
	})

	// Test operations in race conditions.
	t.Run("concurrent", func(t *testing.T) {
		n := time.Now()

		t.Run("task", func(t *testing.T) {
			t.Run("insert", func(t *testing.T) {
				var eg errgroup.Group
				for i := 0; i < 3; i++ {
					eg.Go(func() error {
						_, err := g.InsertTask(ctx, &ratus.Task{
							ID:        "1",
							Topic:     "test",
							State:     ratus.TaskStatePending,
							Produced:  &n,
							Scheduled: &n,
							Payload:   "a",
						})
						return err
					})
				}
				if err := eg.Wait(); !errors.Is(err, ratus.ErrConflict) {
					t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrConflict, err)
				}
				v, err := g.ListTasks(ctx, "test", 10, 0)
				if err != nil {
					t.Error(err)
				}
				if len(v) != 1 {
					t.Errorf("incorrect number of results, expected 1, got %d", len(v))
				}
				d, err := g.DeleteTask(ctx, "1")
				if err != nil {
					t.Error(err)
				}
				if d.Deleted != 1 {
					t.Errorf("incorrect number of deletions, expected 1, got %d", d.Deleted)
				}
			})

			t.Run("upsert", func(t *testing.T) {
				var eg errgroup.Group
				for i := 0; i < 3; i++ {
					eg.Go(func() error {
						_, err := g.UpsertTask(ctx, &ratus.Task{
							ID:        "1",
							Topic:     "test",
							State:     ratus.TaskStatePending,
							Produced:  &n,
							Scheduled: &n,
							Payload:   "a",
						})
						return err
					})
				}
				if err := eg.Wait(); err != nil {
					t.Error(err)
				}
				v, err := g.ListTasks(ctx, "test", 10, 0)
				if err != nil {
					t.Error(err)
				}
				if len(v) != 1 {
					t.Errorf("incorrect number of results, expected 1, got %d", len(v))
				}
				d, err := g.DeleteTasks(ctx, "test")
				if err != nil {
					t.Error(err)
				}
				if d.Deleted != 1 {
					t.Errorf("incorrect number of deletions, expected 1, got %d", d.Deleted)
				}
			})
		})

		t.Run("tasks", func(t *testing.T) {
			t.Run("insert", func(t *testing.T) {
				var (
					eg errgroup.Group
					a  atomic.Int64
				)
				for i := 0; i < 3; i++ {
					eg.Go(func() error {
						u, err := g.InsertTasks(ctx, []*ratus.Task{
							{
								ID:        "1",
								Topic:     "test",
								State:     ratus.TaskStatePending,
								Produced:  &n,
								Scheduled: &n,
								Payload:   "a",
							},
							{
								ID:        "2",
								Topic:     "test",
								State:     ratus.TaskStatePending,
								Produced:  &n,
								Scheduled: &n,
								Payload:   "b",
							},
						})
						a.Add(u.Created)
						return err
					})
				}
				if err := eg.Wait(); err != nil {
					t.Error(err)
				}
				if a.Load() != 2 {
					t.Errorf("incorrect number of creations, expected 2, got %d", a.Load())
				}
				v, err := g.ListTasks(ctx, "test", 10, 0)
				if err != nil {
					t.Error(err)
				}
				if len(v) != 2 {
					t.Errorf("incorrect number of results, expected 2, got %d", len(v))
				}
				d, err := g.DeleteTopic(ctx, "test")
				if err != nil {
					t.Error(err)
				}
				if d.Deleted != 2 {
					t.Errorf("incorrect number of deletions, expected 2, got %d", d.Deleted)
				}
			})

			t.Run("upsert", func(t *testing.T) {
				var eg errgroup.Group
				for i := 0; i < 3; i++ {
					eg.Go(func() error {
						_, err := g.UpsertTasks(ctx, []*ratus.Task{
							{
								ID:        "1",
								Topic:     "test",
								State:     ratus.TaskStatePending,
								Produced:  &n,
								Scheduled: &n,
								Payload:   "a",
							},
							{
								ID:        "2",
								Topic:     "test",
								State:     ratus.TaskStatePending,
								Produced:  &n,
								Scheduled: &n,
								Payload:   "b",
							},
						})
						return err
					})
				}
				if err := eg.Wait(); err != nil {
					t.Error(err)
				}
				v, err := g.ListTasks(ctx, "test", 10, 0)
				if err != nil {
					t.Error(err)
				}
				if len(v) != 2 {
					t.Errorf("incorrect number of results, expected 2, got %d", len(v))
				}
				d, err := g.DeleteTopics(ctx)
				if err != nil {
					t.Error(err)
				}
				if d.Deleted != 2 {
					t.Errorf("incorrect number of deletions, expected 2, got %d", d.Deleted)
				}
			})
		})

		t.Run("promise", func(t *testing.T) {
			t.Run("insert", func(t *testing.T) {
				if _, err := g.InsertTask(ctx, &ratus.Task{
					ID:        "1",
					Topic:     "test",
					State:     ratus.TaskStatePending,
					Produced:  &n,
					Scheduled: &n,
					Payload:   "a",
				}); err != nil {
					t.Error(err)
				}
				var eg errgroup.Group
				for i := 0; i < 3; i++ {
					eg.Go(func() error {
						_, err := g.InsertPromise(ctx, &ratus.Promise{
							ID:       "1",
							Deadline: &n,
						})
						return err
					})
				}
				if err := eg.Wait(); !errors.Is(err, ratus.ErrConflict) {
					t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrConflict, err)
				}
				p, err := g.GetPromise(ctx, "1")
				if err != nil {
					t.Error(err)
				}
				if n.Unix() != p.Deadline.Unix() {
					t.Errorf("incorrect promise deadline, expected %v, got %v", n.Unix(), p.Deadline.Unix())
				}
				v, err := g.GetTask(ctx, "1")
				if err != nil {
					t.Error(err)
				}
				if v.State != ratus.TaskStateActive {
					t.Errorf("incorrect task state, expected %d, got %d", ratus.TaskStateActive, v.State)
				}
				d, err := g.DeletePromise(ctx, "1")
				if err != nil {
					t.Error(err)
				}
				if d.Deleted != 1 {
					t.Errorf("incorrect number of deletions, expected 1, got %d", d.Deleted)
				}
				v, err = g.GetTask(ctx, "1")
				if err != nil {
					t.Error(err)
				}
				if v.State != ratus.TaskStatePending {
					t.Errorf("incorrect task state, expected %d, got %d", ratus.TaskStatePending, v.State)
				}
				d, err = g.DeleteTask(ctx, "1")
				if err != nil {
					t.Error(err)
				}
				if d.Deleted != 1 {
					t.Errorf("incorrect number of deletions, expected 1, got %d", d.Deleted)
				}
			})

			t.Run("upsert", func(t *testing.T) {
				if _, err := g.InsertTask(ctx, &ratus.Task{
					ID:        "1",
					Topic:     "test",
					State:     ratus.TaskStatePending,
					Produced:  &n,
					Scheduled: &n,
					Payload:   "a",
				}); err != nil {
					t.Error(err)
				}
				var eg errgroup.Group
				for i := 0; i < 3; i++ {
					eg.Go(func() error {
						_, err := g.UpsertPromise(ctx, &ratus.Promise{
							ID:       "1",
							Deadline: &n,
						})
						return err
					})
				}
				if err := eg.Wait(); err != nil {
					t.Error(err)
				}
				p, err := g.GetPromise(ctx, "1")
				if err != nil {
					t.Error(err)
				}
				if n.Unix() != p.Deadline.Unix() {
					t.Errorf("incorrect promise deadline, expected %v, got %v", n.Unix(), p.Deadline.Unix())
				}
				v, err := g.GetTask(ctx, "1")
				if err != nil {
					t.Error(err)
				}
				if v.State != ratus.TaskStateActive {
					t.Errorf("incorrect task state, expected %d, got %d", ratus.TaskStateActive, v.State)
				}
				d, err := g.DeletePromises(ctx, "test")
				if err != nil {
					t.Error(err)
				}
				if d.Deleted != 1 {
					t.Errorf("incorrect number of deletions, expected 1, got %d", d.Deleted)
				}
				v, err = g.GetTask(ctx, "1")
				if err != nil {
					t.Error(err)
				}
				if v.State != ratus.TaskStatePending {
					t.Errorf("incorrect task state, expected %d, got %d", ratus.TaskStatePending, v.State)
				}
				d, err = g.DeleteTasks(ctx, "test")
				if err != nil {
					t.Error(err)
				}
				if d.Deleted != 1 {
					t.Errorf("incorrect number of deletions, expected 1, got %d", d.Deleted)
				}
			})
		})

		t.Run("poll", func(t *testing.T) {
			_, err := g.InsertTasks(ctx, []*ratus.Task{
				{
					ID:        "1",
					Topic:     "test",
					State:     ratus.TaskStatePending,
					Produced:  &n,
					Scheduled: &n,
					Payload:   "a",
				},
				{
					ID:        "2",
					Topic:     "test",
					State:     ratus.TaskStatePending,
					Produced:  &n,
					Scheduled: &n,
					Payload:   "b",
				},
			})
			if err != nil {
				t.Error(err)
			}
			var eg errgroup.Group
			for i := 0; i < 3; i++ {
				eg.Go(func() error {
					_, err := g.Poll(ctx, "test", &ratus.Promise{Deadline: &n})
					return err
				})
			}
			if err := eg.Wait(); !errors.Is(err, ratus.ErrNotFound) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrNotFound, err)
			}
			v, err := g.ListPromises(ctx, "test", 10, 0)
			if err != nil {
				t.Error(err)
			}
			if len(v) != 2 {
				t.Errorf("incorrect number of results, expected 2, got %d", len(v))
			}
			if err := g.Chore(ctx); err != nil {
				t.Error(err)
			}
			v, err = g.ListPromises(ctx, "test", 10, 0)
			if err != nil {
				t.Error(err)
			}
			if len(v) != 0 {
				t.Errorf("incorrect number of results, expected 0, got %d", len(v))
			}
			d, err := g.DeleteTopic(ctx, "test")
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 2 {
				t.Errorf("incorrect number of deletions, expected 2, got %d", d.Deleted)
			}
		})

		t.Run("commit", func(t *testing.T) {
			k := nonce.Generate(ratus.NonceLength)
			s := ratus.TaskStateArchived
			m := &ratus.Commit{
				Nonce:     k,
				Topic:     "archived",
				State:     &s,
				Scheduled: &n,
				Payload:   "archived",
			}
			if _, err := g.InsertTask(ctx, &ratus.Task{
				ID:        "1",
				Topic:     "test",
				State:     ratus.TaskStatePending,
				Nonce:     k,
				Produced:  &n,
				Scheduled: &n,
				Payload:   "a",
			}); err != nil {
				t.Error(err)
			}
			var eg errgroup.Group
			for i := 0; i < 3; i++ {
				eg.Go(func() error {
					_, err := g.Commit(ctx, "1", m)
					return err
				})
			}
			if err := eg.Wait(); !errors.Is(err, ratus.ErrConflict) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrConflict, err)
			}
			v, err := g.GetTask(ctx, "1")
			if err != nil {
				t.Error(err)
			}
			if v.State != ratus.TaskStateArchived {
				t.Errorf("incorrect task state, expected %d, got %d", ratus.TaskStateArchived, v.State)
			}
			if fmt.Sprint(v.Payload) != "archived" {
				t.Errorf("incorrect payload in task, expected %q, got %q", "archived", v.Payload)
			}
			d, err := g.DeleteTasks(ctx, "archived")
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 1 {
				t.Errorf("incorrect number of deletions, expected 1, got %d", d.Deleted)
			}
		})

		t.Run("clean", func(t *testing.T) {
			d, err := g.DeleteTopics(ctx)
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 0 {
				t.Errorf("incorrect number of deletions, expected 0, got %d", d.Deleted)
			}
		})
	})

	// Test operations related to task scheduling.
	t.Run("schedule", func(t *testing.T) {
		n := time.Now()
		n1 := n.Add(100 * time.Millisecond)
		n2 := n.Add(200 * time.Millisecond)
		if _, err := g.InsertTasks(ctx, []*ratus.Task{
			{
				ID:        "1",
				Topic:     "test",
				State:     ratus.TaskStatePending,
				Produced:  &n,
				Scheduled: &n,
				Payload:   "a",
			},
			{
				ID:        "2",
				Topic:     "test",
				State:     ratus.TaskStatePending,
				Produced:  &n,
				Scheduled: &n1,
				Payload:   "b",
			},
		}); err != nil {
			t.Error(err)
		}

		t.Run("poll", func(t *testing.T) {
			v, err := g.Poll(ctx, "test", &ratus.Promise{Deadline: &n1})
			if err != nil {
				t.Error(err)
			}
			if v.ID != "1" {
				t.Errorf("incorrect task order, expected %q, got %q", "1", v.ID)
			}
			if _, err := g.Poll(ctx, "test", &ratus.Promise{Deadline: &n1}); !errors.Is(err, ratus.ErrNotFound) {
				t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrNotFound, err)
			}
			time.Sleep(100 * time.Millisecond)
			v, err = g.Poll(ctx, "test", &ratus.Promise{Deadline: &n2})
			if err != nil {
				t.Error(err)
			}
			if v.ID != "2" {
				t.Errorf("incorrect task order, expected %q, got %q", "2", v.ID)
			}
		})

		t.Run("clean", func(t *testing.T) {
			d, err := g.DeleteTopics(ctx)
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 2 {
				t.Errorf("incorrect number of deletions, expected 2, got %d", d.Deleted)
			}
		})
	})

	// Test operations with pagination support.
	t.Run("pagination", func(t *testing.T) {
		n := time.Now()
		if _, err := g.InsertTasks(ctx, []*ratus.Task{
			{
				ID:        "1",
				Topic:     "a",
				State:     ratus.TaskStateActive,
				Produced:  &n,
				Scheduled: &n,
				Payload:   "a",
			},
			{
				ID:        "2",
				Topic:     "b",
				State:     ratus.TaskStateActive,
				Produced:  &n,
				Scheduled: &n,
				Payload:   "b",
			},
			{
				ID:        "3",
				Topic:     "c",
				State:     ratus.TaskStateActive,
				Produced:  &n,
				Scheduled: &n,
				Payload:   "c-3",
			},
			{
				ID:        "4",
				Topic:     "c",
				State:     ratus.TaskStateActive,
				Produced:  &n,
				Scheduled: &n,
				Payload:   "c-4",
			},
		}); err != nil {
			t.Error(err)
		}

		t.Run("topic", func(t *testing.T) {
			v, err := g.ListTopics(ctx, 1, 1)
			if err != nil {
				t.Error(err)
			}
			if len(v) != 1 {
				t.Errorf("incorrect number of results, expected 1, got %d", len(v))
			}
			v, err = g.ListTopics(ctx, 10, 10)
			if err != nil {
				t.Error(err)
			}
			if len(v) != 0 {
				t.Errorf("incorrect number of results, expected 0, got %d", len(v))
			}
		})

		t.Run("task", func(t *testing.T) {
			v, err := g.ListTasks(ctx, "c", 1, 1)
			if err != nil {
				t.Error(err)
			}
			if len(v) != 1 {
				t.Errorf("incorrect number of results, expected 1, got %d", len(v))
			}
			v, err = g.ListTasks(ctx, "c", 10, 10)
			if err != nil {
				t.Error(err)
			}
			if len(v) != 0 {
				t.Errorf("incorrect number of results, expected 0, got %d", len(v))
			}
		})

		t.Run("promise", func(t *testing.T) {
			v, err := g.ListPromises(ctx, "c", 1, 1)
			if err != nil {
				t.Error(err)
			}
			if len(v) != 1 {
				t.Errorf("incorrect number of results, expected 1, got %d", len(v))
			}
			v, err = g.ListPromises(ctx, "c", 10, 10)
			if err != nil {
				t.Error(err)
			}
			if len(v) != 0 {
				t.Errorf("incorrect number of results, expected 0, got %d", len(v))
			}
		})

		t.Run("clean", func(t *testing.T) {
			d, err := g.DeleteTopics(ctx)
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 4 {
				t.Errorf("incorrect number of deletions, expected 4, got %d", d.Deleted)
			}
		})
	})

	// Test operations that encode and decode payloads.
	t.Run("payload", func(t *testing.T) {
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
				if _, err := g.InsertTask(ctx, p.task); err != nil {
					t.Error(err)
				}
				v, err := g.GetTask(ctx, p.task.ID)
				if err != nil {
					t.Error()
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

		t.Run("clean", func(t *testing.T) {
			d, err := g.DeleteTopics(ctx)
			if err != nil {
				t.Error(err)
			}
			if d.Deleted != 7 {
				t.Errorf("incorrect number of deletions, expected 7, got %d", d.Deleted)
			}
		})
	})
}
