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

		t.Run("topics", func(t *testing.T) {
			t.Parallel()

			t.Run("get", func(t *testing.T) {
				t.Parallel()
				v, err := client.ListTopics(ctx, 10, 0)
				if err != nil {
					t.Error(err)
				}
				if len(v) == 0 {
					t.Fail()
				}
			})

			t.Run("delete", func(t *testing.T) {
				t.Parallel()
				v, err := client.DeleteTopics(ctx)
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.Deleted == 0 {
					t.Fail()
				}
			})
		})

		t.Run("topic", func(t *testing.T) {
			t.Parallel()

			t.Run("get", func(t *testing.T) {
				t.Parallel()
				v, err := client.GetTopic(ctx, "topic")
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.Count == 0 {
					t.Fail()
				}
			})

			t.Run("delete", func(t *testing.T) {
				t.Parallel()
				v, err := client.DeleteTopic(ctx, "topic")
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.Deleted == 0 {
					t.Fail()
				}
			})
		})

		t.Run("tasks", func(t *testing.T) {
			t.Parallel()

			t.Run("get", func(t *testing.T) {
				t.Parallel()
				v, err := client.ListTasks(ctx, "topic", 10, 0)
				if err != nil {
					t.Error(err)
				}
				if len(v) == 0 {
					t.Fail()
				}
			})

			t.Run("post", func(t *testing.T) {
				t.Parallel()
				v, err := client.InsertTasks(ctx, []*ratus.Task{{ID: "id", Topic: "topic"}})
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.Created == 0 {
					t.Fail()
				}
			})

			t.Run("put", func(t *testing.T) {
				t.Parallel()
				v, err := client.UpsertTasks(ctx, []*ratus.Task{{ID: "id", Topic: "topic"}})
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.Updated == 0 {
					t.Fail()
				}
			})

			t.Run("delete", func(t *testing.T) {
				t.Parallel()
				v, err := client.DeleteTasks(ctx, "topic")
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.Deleted == 0 {
					t.Fail()
				}
			})
		})

		t.Run("task", func(t *testing.T) {
			t.Parallel()

			t.Run("get", func(t *testing.T) {
				t.Parallel()
				v, err := client.GetTask(ctx, "id")
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.ID != "id" {
					t.Fail()
				}
			})

			t.Run("post", func(t *testing.T) {
				t.Parallel()
				v, err := client.InsertTask(ctx, &ratus.Task{ID: "id", Topic: "topic"})
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.Created == 0 {
					t.Fail()
				}
			})

			t.Run("put", func(t *testing.T) {
				t.Parallel()
				v, err := client.UpsertTask(ctx, &ratus.Task{ID: "id", Topic: "topic"})
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.Updated == 0 {
					t.Fail()
				}
			})

			t.Run("delete", func(t *testing.T) {
				t.Parallel()
				v, err := client.DeleteTask(ctx, "id")
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.Deleted == 0 {
					t.Fail()
				}
			})

			t.Run("patch", func(t *testing.T) {
				t.Parallel()
				v, err := client.PatchTask(ctx, "id", &ratus.Commit{})
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.ID != "id" {
					t.Fail()
				}
			})
		})

		t.Run("promises", func(t *testing.T) {
			t.Parallel()

			t.Run("get", func(t *testing.T) {
				t.Parallel()
				v, err := client.ListPromises(ctx, "topic", 10, 0)
				if err != nil {
					t.Error(err)
				}
				if len(v) == 0 {
					t.Fail()
				}
			})

			t.Run("post", func(t *testing.T) {
				t.Parallel()
				v, err := client.PostPromises(ctx, "topic", &ratus.Promise{})
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.Topic != "topic" {
					t.Fail()
				}
			})

			t.Run("delete", func(t *testing.T) {
				t.Parallel()
				v, err := client.DeletePromises(ctx, "topic")
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.Deleted == 0 {
					t.Fail()
				}
			})
		})

		t.Run("promise", func(t *testing.T) {
			t.Parallel()

			t.Run("get", func(t *testing.T) {
				t.Parallel()
				v, err := client.GetPromise(ctx, "id")
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.ID != "id" {
					t.Fail()
				}
			})

			t.Run("post", func(t *testing.T) {
				t.Parallel()
				v, err := client.InsertPromise(ctx, &ratus.Promise{ID: "id"})
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.State != ratus.TaskStateActive {
					t.Fail()
				}
			})

			t.Run("put", func(t *testing.T) {
				t.Parallel()
				v, err := client.UpsertPromise(ctx, &ratus.Promise{ID: "id"})
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.State != ratus.TaskStateActive {
					t.Fail()
				}
			})

			t.Run("delete", func(t *testing.T) {
				t.Parallel()
				v, err := client.DeletePromise(ctx, "id")
				if err != nil {
					t.Error(err)
				}
				if v == nil || v.Deleted == 0 {
					t.Fail()
				}
			})
		})

		t.Run("health", func(t *testing.T) {
			t.Parallel()

			t.Run("livez", func(t *testing.T) {
				t.Parallel()
				if err := client.GetLiveness(ctx); err != nil {
					t.Error(err)
				}
			})

			t.Run("readyz", func(t *testing.T) {
				t.Parallel()
				if err := client.GetReadiness(ctx); err != nil {
					t.Error(err)
				}
			})
		})
	})

	t.Run("unavailable", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		client := newClient(t, &stub.Engine{Err: ratus.ErrServiceUnavailable})
		for _, f := range []func() (any, error){
			func() (any, error) { return client.Poll(ctx, "topic", &ratus.Promise{Timeout: "30s"}) },
			func() (any, error) { return client.ListTopics(ctx, 10, 0) },
			func() (any, error) { return client.DeleteTopics(ctx) },
			func() (any, error) { return client.GetTopic(ctx, "topic") },
			func() (any, error) { return client.DeleteTopic(ctx, "topic") },
			func() (any, error) { return client.ListTasks(ctx, "topic", 10, 0) },
			func() (any, error) { return client.InsertTasks(ctx, []*ratus.Task{{ID: "id", Topic: "topic"}}) },
			func() (any, error) { return client.UpsertTasks(ctx, []*ratus.Task{{ID: "id", Topic: "topic"}}) },
			func() (any, error) { return client.DeleteTasks(ctx, "topic") },
			func() (any, error) { return client.GetTask(ctx, "id") },
			func() (any, error) { return client.InsertTask(ctx, &ratus.Task{ID: "id", Topic: "topic"}) },
			func() (any, error) { return client.UpsertTask(ctx, &ratus.Task{ID: "id", Topic: "topic"}) },
			func() (any, error) { return client.DeleteTask(ctx, "id") },
			func() (any, error) { return client.PatchTask(ctx, "id", &ratus.Commit{}) },
			func() (any, error) { return client.ListPromises(ctx, "topic", 10, 0) },
			func() (any, error) { return client.PostPromises(ctx, "topic", &ratus.Promise{}) },
			func() (any, error) { return client.DeletePromises(ctx, "topic") },
			func() (any, error) { return client.GetPromise(ctx, "id") },
			func() (any, error) { return client.InsertPromise(ctx, &ratus.Promise{ID: "id"}) },
			func() (any, error) { return client.UpsertPromise(ctx, &ratus.Promise{ID: "id"}) },
			func() (any, error) { return client.DeletePromise(ctx, "id") },
			func() (any, error) { return nil, client.GetReadiness(ctx) },
		} {
			if _, err := f(); !errors.Is(err, ratus.ErrServiceUnavailable) {
				t.Fail()
			}
		}
	})

	t.Run("event", func(t *testing.T) {
		t.Parallel()

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

		t.Run("error", func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			defer cancel()

			client := newClient(t, &stub.Engine{Err: ratus.ErrInternalServerError})

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
	})

	t.Run("invalid", func(t *testing.T) {
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
