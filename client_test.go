package ratus_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/config"
	"github.com/hyperonym/ratus/internal/controller"
	"github.com/hyperonym/ratus/internal/engine/stub"
	"github.com/hyperonym/ratus/internal/middleware"
	"github.com/hyperonym/ratus/internal/router"
)

func newClient(t *testing.T, g *stub.Engine) *ratus.Client {
	t.Helper()
	o := config.PaginationConfig{MaxLimit: 10, MaxOffset: 10}
	r := router.New(&controller.V1{
		Pagination: middleware.Pagination(&o),
		Topic:      controller.NewTopicController(g),
		Task:       controller.NewTaskController(g),
		Promise:    controller.NewPromiseController(g),
		Health:     controller.NewHealthController(g),
		Metrics:    controller.NewMetricsController(g),
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
	gin.SetMode(gin.ReleaseMode)

	t.Run("normal", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		client := newClient(t, &stub.Engine{})

		t.Run("subscribe", func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
			defer cancel()

			var a atomic.Int32
			if err := client.Subscribe(ctx, &ratus.SubscribeOptions{
				Promise:      &ratus.Promise{Timeout: "30s"},
				Topic:        "topic",
				PollInterval: 100 * time.Millisecond,
			}, func(c *ratus.Context, err error) {
				if err != nil {
					t.Error(err)
					return
				}
				a.Add(1)
				if err := c.Commit(); err != nil {
					t.Error(err)
				}
			}); !errors.Is(err, context.DeadlineExceeded) {
				t.Error(err)
			}
			if a.Load() != 2 {
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
		client := newClient(t, &stub.Engine{Err: ratus.ErrServiceUnavailable})

		t.Run("subscribe", func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
			defer cancel()

			var a atomic.Int32
			if err := client.Subscribe(ctx, &ratus.SubscribeOptions{
				Promise:       &ratus.Promise{Timeout: "30s"},
				Topic:         "topic",
				ErrorInterval: time.Duration(100 * time.Millisecond),
			}, func(c *ratus.Context, err error) {
				if err == nil {
					t.Fail()
				}
				a.Add(1)
			}); !errors.Is(err, context.DeadlineExceeded) {
				t.Error(err)
			}
			if a.Load() != 2 {
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
			client := newClient(t, &stub.Engine{})

			var a atomic.Int32
			if err := client.Subscribe(ctx, &ratus.SubscribeOptions{
				Promise:       &ratus.Promise{Timeout: "30s"},
				Topic:         "topic",
				ErrorInterval: time.Duration(100 * time.Millisecond),
			}, func(c *ratus.Context, err error) {
				if err != nil {
					t.Error(err)
					return
				}
				a.Add(1)
				cancel()
			}); !errors.Is(err, context.Canceled) {
				t.Error(err)
			}
			if a.Load() != 1 {
				t.Fail()
			}
		})

		t.Run("drain", func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			defer cancel()

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Header().Add("Content-Type", "application/json")
				e := ratus.NewError(ratus.ErrNotFound)
				b, _ := json.Marshal(e)
				fmt.Fprintln(w, string(b))
			}))
			defer ts.Close()

			client, err := ratus.NewClient(&ratus.ClientOptions{Origin: ts.URL})
			if err != nil {
				t.Error(err)
			}

			var a atomic.Int32
			if err := client.Subscribe(ctx, &ratus.SubscribeOptions{
				Promise:          &ratus.Promise{Timeout: "30s"},
				Topic:            "topic",
				Concurrency:      1,
				ConcurrencyDelay: 1 * time.Microsecond,
				PollInterval:     1 * time.Millisecond,
				DrainInterval:    100 * time.Millisecond,
				ErrorInterval:    1 * time.Millisecond,
			}, func(c *ratus.Context, err error) {
				if err != nil {
					t.Error(err)
					return
				}
				a.Add(1)
			}); !errors.Is(err, context.DeadlineExceeded) {
				t.Error(err)
			}
			if a.Load() != 0 {
				t.Fail()
			}
		})

		t.Run("request", func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			client := newClient(t, &stub.Engine{})

			t.Run("method", func(t *testing.T) {
				t.Parallel()
				if err := client.Request(ctx, "?", "/", nil, nil); err == nil {
					t.Fail()
				}
			})

			t.Run("body", func(t *testing.T) {
				t.Parallel()
				if err := client.Request(ctx, "POST", "/", func() {}, nil); err == nil {
					t.Fail()
				}
			})
		})

		t.Run("response", func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Header().Add("Content-Type", "application/json")
				fmt.Fprintln(w, ".")
			}))
			defer ts.Close()

			client, err := ratus.NewClient(&ratus.ClientOptions{Origin: ts.URL})
			if err != nil {
				t.Error(err)
			}

			if err := client.Request(ctx, "GET", "/", nil, nil); err == nil {
				t.Fail()
			}
		})
	})
}
