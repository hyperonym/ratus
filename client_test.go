package ratus_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/config"
	"github.com/hyperonym/ratus/internal/controller"
	"github.com/hyperonym/ratus/internal/engine/stub"
	"github.com/hyperonym/ratus/internal/middleware"
	"github.com/hyperonym/ratus/internal/router"
)

func newClient(t *testing.T, err error) *ratus.Client {
	t.Helper()
	g := stub.Engine{Err: err}
	o := config.PaginationConfig{MaxLimit: 10, MaxOffset: 10}
	r := router.New(&controller.V1{
		Pagination: middleware.Pagination(&o),
		Topic:      controller.NewTopicController(&g),
		Task:       controller.NewTaskController(&g),
		Promise:    controller.NewPromiseController(&g),
		Health:     controller.NewHealthController(&g),
		Metrics:    controller.NewMetricsController(&g),
	})
	ts := httptest.NewServer(r.Handler())
	t.Cleanup(func() {
		ts.Close()
	})
	c, err := ratus.NewClient(&ratus.ClientOptions{
		Origin:  ts.URL,
		Headers: map[string]string{"foo": "bar"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestClient(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		client := newClient(t, nil)

		t.Run("subscribe", func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			var a atomic.Int32
			if err := client.Subscribe(ctx, &ratus.SubscribeOptions{
				Promise: &ratus.Promise{Timeout: "30s"},
				Topic:   "topic",
			}, func(c *ratus.Context, err error) {
				if err != nil {
					t.Error(err)
					return
				}
				a.Add(1)
				if err := c.Commit(); err != nil {
					t.Error(err)
				}
				time.Sleep(1 * time.Second)
			}); !errors.Is(err, context.DeadlineExceeded) {
				t.Error(err)
			}
			if a.Load() <= 0 {
				t.Fail()
			}
		})

		t.Run("poll", func(t *testing.T) {
			t.Parallel()
			c, err := client.Poll(ctx, "topic", &ratus.Promise{Timeout: "30s"})
			if err != nil {
				t.Error(err)
			}
			if err := c.Commit(); err != nil {
				t.Error(err)
			}
		})

		for _, x := range []struct {
			name string
			test func() (any, error)
		}{
			{"list-topics", func() (any, error) { return client.ListTopics(ctx, 10, 0) }},
			{"delete-topics", func() (any, error) { return client.DeleteTopics(ctx) }},
			{"get-topic", func() (any, error) { return client.GetTopic(ctx, "topic") }},
			{"delete-topic", func() (any, error) { return client.DeleteTopic(ctx, "topic") }},
			{"list-tasks", func() (any, error) { return client.ListTasks(ctx, "topic", 10, 0) }},
			{"insert-tasks", func() (any, error) { return client.InsertTasks(ctx, []*ratus.Task{{ID: "id", Topic: "topic"}}) }},
			{"upsert-tasks", func() (any, error) { return client.UpsertTasks(ctx, []*ratus.Task{{ID: "id", Topic: "topic"}}) }},
			{"delete-tasks", func() (any, error) { return client.DeleteTasks(ctx, "topic") }},
			{"get-task", func() (any, error) { return client.GetTask(ctx, "id") }},
			{"insert-task", func() (any, error) { return client.InsertTask(ctx, &ratus.Task{ID: "id", Topic: "topic"}) }},
			{"upsert-task", func() (any, error) { return client.UpsertTask(ctx, &ratus.Task{ID: "id", Topic: "topic"}) }},
			{"delete-task", func() (any, error) { return client.DeleteTask(ctx, "id") }},
			{"patch-task", func() (any, error) { return client.PatchTask(ctx, "id", &ratus.Commit{}) }},
			{"list-promises", func() (any, error) { return client.ListPromises(ctx, "topic", 10, 0) }},
			{"post-promises", func() (any, error) { return client.PostPromises(ctx, "topic", &ratus.Promise{}) }},
			{"delete-promises", func() (any, error) { return client.DeletePromises(ctx, "topic") }},
			{"get-promise", func() (any, error) { return client.GetPromise(ctx, "id") }},
			{"insert-promise", func() (any, error) { return client.InsertPromise(ctx, &ratus.Promise{ID: "id"}) }},
			{"upsert-promise", func() (any, error) { return client.UpsertPromise(ctx, &ratus.Promise{ID: "id"}) }},
			{"delete-promise", func() (any, error) { return client.DeletePromise(ctx, "id") }},
			{"get-liveness", func() (any, error) { return nil, client.GetLiveness(ctx) }},
			{"get-readiness", func() (any, error) { return nil, client.GetReadiness(ctx) }},
		} {
			q := x
			t.Run(q.name, func(t *testing.T) {
				t.Parallel()
				if _, err := q.test(); err != nil {
					t.Error(err)
				}
			})
		}
	})

	t.Run("unavailable", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		client := newClient(t, ratus.ErrServiceUnavailable)

		t.Run("subscribe", func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			var a atomic.Int32
			if err := client.Subscribe(ctx, &ratus.SubscribeOptions{
				Promise:       &ratus.Promise{Timeout: "30s"},
				Topic:         "topic",
				ErrorInterval: time.Duration(1 * time.Second),
			}, func(c *ratus.Context, err error) {
				if err == nil {
					t.Fail()
				}
				a.Add(1)
			}); !errors.Is(err, context.DeadlineExceeded) {
				t.Error(err)
			}
			if a.Load() <= 0 {
				t.Fail()
			}
		})

		for _, x := range []struct {
			name string
			test func() (any, error)
		}{
			{"poll", func() (any, error) { return client.Poll(ctx, "topic", &ratus.Promise{Timeout: "30s"}) }},
			{"list-topics", func() (any, error) { return client.ListTopics(ctx, 10, 0) }},
			{"delete-topics", func() (any, error) { return client.DeleteTopics(ctx) }},
			{"get-topic", func() (any, error) { return client.GetTopic(ctx, "topic") }},
			{"delete-topic", func() (any, error) { return client.DeleteTopic(ctx, "topic") }},
			{"list-tasks", func() (any, error) { return client.ListTasks(ctx, "topic", 10, 0) }},
			{"insert-tasks", func() (any, error) { return client.InsertTasks(ctx, []*ratus.Task{{ID: "id", Topic: "topic"}}) }},
			{"upsert-tasks", func() (any, error) { return client.UpsertTasks(ctx, []*ratus.Task{{ID: "id", Topic: "topic"}}) }},
			{"delete-tasks", func() (any, error) { return client.DeleteTasks(ctx, "topic") }},
			{"get-task", func() (any, error) { return client.GetTask(ctx, "id") }},
			{"insert-task", func() (any, error) { return client.InsertTask(ctx, &ratus.Task{ID: "id", Topic: "topic"}) }},
			{"upsert-task", func() (any, error) { return client.UpsertTask(ctx, &ratus.Task{ID: "id", Topic: "topic"}) }},
			{"delete-task", func() (any, error) { return client.DeleteTask(ctx, "id") }},
			{"patch-task", func() (any, error) { return client.PatchTask(ctx, "id", &ratus.Commit{}) }},
			{"list-promises", func() (any, error) { return client.ListPromises(ctx, "topic", 10, 0) }},
			{"post-promises", func() (any, error) { return client.PostPromises(ctx, "topic", &ratus.Promise{}) }},
			{"delete-promises", func() (any, error) { return client.DeletePromises(ctx, "topic") }},
			{"get-promise", func() (any, error) { return client.GetPromise(ctx, "id") }},
			{"insert-promise", func() (any, error) { return client.InsertPromise(ctx, &ratus.Promise{ID: "id"}) }},
			{"upsert-promise", func() (any, error) { return client.UpsertPromise(ctx, &ratus.Promise{ID: "id"}) }},
			{"delete-promise", func() (any, error) { return client.DeletePromise(ctx, "id") }},
			{"get-readiness", func() (any, error) { return nil, client.GetReadiness(ctx) }},
		} {
			q := x
			t.Run(q.name, func(t *testing.T) {
				t.Parallel()
				if _, err := q.test(); !errors.Is(err, ratus.ErrServiceUnavailable) {
					t.Fail()
				}
			})
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		t.Run("origin", func(t *testing.T) {
			t.Parallel()
			if _, err := ratus.NewClient(&ratus.ClientOptions{Origin: "*://*://"}); err == nil {
				t.Fail()
			}
		})

		t.Run("context", func(t *testing.T) {
			t.Parallel()
			var c ratus.Context
			c.Task = &ratus.Task{}
			c.SetNonce("")
			c.SetTopic("")
			c.SetState(ratus.TaskStatePending)
			c.SetScheduled(time.Now())
			c.SetPayload("")
			c.SetDefer("")
			c.Force()
			c.Abstain()
			c.Archive()
			c.Reschedule(time.Now())
			c.Retry("")
			c.Reset()
			if err := c.Commit(); err == nil {
				t.Fail()
			}
		})

		t.Run("cancel", func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			client := newClient(t, nil)

			var a atomic.Int32
			if err := client.Subscribe(ctx, &ratus.SubscribeOptions{
				Promise:       &ratus.Promise{Timeout: "30s"},
				Topic:         "topic",
				ErrorInterval: time.Duration(1 * time.Second),
			}, func(c *ratus.Context, err error) {
				a.Add(1)
				cancel()
			}); !errors.Is(err, context.Canceled) {
				t.Error(err)
			}
			if a.Load() != 2 {
				t.Fail()
			}
		})
	})
}
