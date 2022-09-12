package ratus_test

import (
	"context"
	"encoding/json"
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

func TestClient(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		o := config.PaginationConfig{MaxLimit: 10, MaxOffset: 10}
		g := stub.Engine{Err: nil}
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

		client, err := ratus.NewClient(&ratus.ClientOptions{
			Origin:  ts.URL,
			Headers: map[string]string{"foo": "bar"},
		})
		if err != nil {
			t.Error(err)
		}

		t.Run("subscribe", func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			var a atomic.Int32
			err := client.Subscribe(ctx, &ratus.SubscribeOptions{
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
			})
			if !errors.Is(err, context.DeadlineExceeded) {
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

		t.Run("list-topics", func(t *testing.T) {
			t.Parallel()
			v, err := client.ListTopics(ctx, 10, 0)
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("delete-topics", func(t *testing.T) {
			t.Parallel()
			v, err := client.DeleteTopics(ctx)
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("get-topic", func(t *testing.T) {
			t.Parallel()
			v, err := client.GetTopic(ctx, "topic")
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("delete-topic", func(t *testing.T) {
			t.Parallel()
			v, err := client.DeleteTopic(ctx, "topic")
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("list-tasks", func(t *testing.T) {
			t.Parallel()
			v, err := client.ListTasks(ctx, "topic", 10, 0)
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("insert-tasks", func(t *testing.T) {
			t.Parallel()
			v, err := client.InsertTasks(ctx, []*ratus.Task{{ID: "id", Topic: "topic"}})
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("upsert-tasks", func(t *testing.T) {
			t.Parallel()
			v, err := client.UpsertTasks(ctx, []*ratus.Task{{ID: "id", Topic: "topic"}})
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("delete-tasks", func(t *testing.T) {
			t.Parallel()
			v, err := client.DeleteTasks(ctx, "topic")
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("get-task", func(t *testing.T) {
			t.Parallel()
			v, err := client.GetTask(ctx, "id")
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("insert-task", func(t *testing.T) {
			t.Parallel()
			v, err := client.InsertTask(ctx, &ratus.Task{ID: "id", Topic: "topic"})
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("upsert-task", func(t *testing.T) {
			t.Parallel()
			v, err := client.UpsertTask(ctx, &ratus.Task{ID: "id", Topic: "topic"})
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("delete-task", func(t *testing.T) {
			t.Parallel()
			v, err := client.DeleteTask(ctx, "id")
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("patch-task", func(t *testing.T) {
			t.Parallel()
			v, err := client.PatchTask(ctx, "id", &ratus.Commit{})
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("list-promises", func(t *testing.T) {
			t.Parallel()
			v, err := client.ListPromises(ctx, "topic", 10, 0)
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("post-promises", func(t *testing.T) {
			t.Parallel()
			v, err := client.PostPromises(ctx, "topic", &ratus.Promise{})
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("delete-promises", func(t *testing.T) {
			t.Parallel()
			v, err := client.DeletePromises(ctx, "topic")
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("get-promise", func(t *testing.T) {
			t.Parallel()
			v, err := client.GetPromise(ctx, "id")
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("insert-promise", func(t *testing.T) {
			t.Parallel()
			v, err := client.InsertPromise(ctx, &ratus.Promise{ID: "id"})
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("upsert-promise", func(t *testing.T) {
			t.Parallel()
			v, err := client.UpsertPromise(ctx, &ratus.Promise{ID: "id"})
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("delete-promise", func(t *testing.T) {
			t.Parallel()
			v, err := client.DeletePromise(ctx, "id")
			if err != nil {
				t.Error(err)
			}
			assertExist(t, v)
		})

		t.Run("get-liveness", func(t *testing.T) {
			t.Parallel()
			if err := client.GetLiveness(ctx); err != nil {
				t.Error(err)
			}
		})

		t.Run("get-readiness", func(t *testing.T) {
			t.Parallel()
			if err := client.GetReadiness(ctx); err != nil {
				t.Error(err)
			}
		})
	})

	t.Run("unavailable", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		o := config.PaginationConfig{MaxLimit: 10, MaxOffset: 10}
		g := stub.Engine{Err: ratus.ErrServiceUnavailable}
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

		client, err := ratus.NewClient(&ratus.ClientOptions{
			Origin:  ts.URL,
			Headers: map[string]string{"foo": "bar"},
		})
		if err != nil {
			t.Error(err)
		}

		t.Run("subscribe", func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			var a atomic.Int32
			err := client.Subscribe(ctx, &ratus.SubscribeOptions{
				Promise:       &ratus.Promise{Timeout: "30s"},
				Topic:         "topic",
				ErrorInterval: time.Duration(1 * time.Second),
			}, func(c *ratus.Context, err error) {
				if err == nil {
					t.Fail()
				}
				a.Add(1)
			})
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Error(err)
			}
			if a.Load() <= 0 {
				t.Fail()
			}
		})

		t.Run("poll", func(t *testing.T) {
			t.Parallel()
			_, err := client.Poll(ctx, "topic", &ratus.Promise{Timeout: "30s"})
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("list-topics", func(t *testing.T) {
			t.Parallel()
			_, err := client.ListTopics(ctx, 10, 0)
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("delete-topics", func(t *testing.T) {
			t.Parallel()
			_, err := client.DeleteTopics(ctx)
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("get-topic", func(t *testing.T) {
			t.Parallel()
			_, err := client.GetTopic(ctx, "topic")
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("delete-topic", func(t *testing.T) {
			t.Parallel()
			_, err := client.DeleteTopic(ctx, "topic")
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("list-tasks", func(t *testing.T) {
			t.Parallel()
			_, err := client.ListTasks(ctx, "topic", 10, 0)
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("insert-tasks", func(t *testing.T) {
			t.Parallel()
			_, err := client.InsertTasks(ctx, []*ratus.Task{{ID: "id", Topic: "topic"}})
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("upsert-tasks", func(t *testing.T) {
			t.Parallel()
			_, err := client.UpsertTasks(ctx, []*ratus.Task{{ID: "id", Topic: "topic"}})
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("delete-tasks", func(t *testing.T) {
			t.Parallel()
			_, err := client.DeleteTasks(ctx, "topic")
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("get-task", func(t *testing.T) {
			t.Parallel()
			_, err := client.GetTask(ctx, "id")
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("insert-task", func(t *testing.T) {
			t.Parallel()
			_, err := client.InsertTask(ctx, &ratus.Task{ID: "id", Topic: "topic"})
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("upsert-task", func(t *testing.T) {
			t.Parallel()
			_, err := client.UpsertTask(ctx, &ratus.Task{ID: "id", Topic: "topic"})
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("delete-task", func(t *testing.T) {
			t.Parallel()
			_, err := client.DeleteTask(ctx, "id")
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("patch-task", func(t *testing.T) {
			t.Parallel()
			_, err := client.PatchTask(ctx, "id", &ratus.Commit{})
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("list-promises", func(t *testing.T) {
			t.Parallel()
			_, err := client.ListPromises(ctx, "topic", 10, 0)
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("post-promises", func(t *testing.T) {
			t.Parallel()
			_, err := client.PostPromises(ctx, "topic", &ratus.Promise{})
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("delete-promises", func(t *testing.T) {
			t.Parallel()
			_, err := client.DeletePromises(ctx, "topic")
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("get-promise", func(t *testing.T) {
			t.Parallel()
			_, err := client.GetPromise(ctx, "id")
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("insert-promise", func(t *testing.T) {
			t.Parallel()
			_, err := client.InsertPromise(ctx, &ratus.Promise{ID: "id"})
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("upsert-promise", func(t *testing.T) {
			t.Parallel()
			_, err := client.UpsertPromise(ctx, &ratus.Promise{ID: "id"})
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("delete-promise", func(t *testing.T) {
			t.Parallel()
			_, err := client.DeletePromise(ctx, "id")
			assertError(t, err, ratus.ErrServiceUnavailable)
		})

		t.Run("get-readiness", func(t *testing.T) {
			t.Parallel()
			err := client.GetReadiness(ctx)
			assertError(t, err, ratus.ErrServiceUnavailable)
		})
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
			o := config.PaginationConfig{MaxLimit: 10, MaxOffset: 10}
			g := stub.Engine{Err: nil}
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

			client, err := ratus.NewClient(&ratus.ClientOptions{Origin: ts.URL})
			if err != nil {
				t.Error(err)
			}

			var a atomic.Int32
			err = client.Subscribe(ctx, &ratus.SubscribeOptions{
				Promise:       &ratus.Promise{Timeout: "30s"},
				Topic:         "topic",
				ErrorInterval: time.Duration(1 * time.Second),
			}, func(c *ratus.Context, err error) {
				a.Add(1)
				cancel()
			})
			if !errors.Is(err, context.Canceled) {
				t.Error(err)
			}
			if a.Load() != 2 {
				t.Fail()
			}
		})
	})
}
