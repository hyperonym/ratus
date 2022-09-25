package controller_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/config"
	"github.com/hyperonym/ratus/internal/controller"
	"github.com/hyperonym/ratus/internal/engine/stub"
	"github.com/hyperonym/ratus/internal/middleware"
	"github.com/hyperonym/ratus/internal/reqtest"
)

func TestController(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	t.Run("v1", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()
			o := config.PaginationConfig{MaxLimit: 10, MaxOffset: 10}
			g := stub.Engine{Err: nil}
			h := reqtest.NewHandler(&controller.V1{
				Pagination: middleware.Pagination(&o),
				Topic:      controller.NewTopicController(&g),
				Task:       controller.NewTaskController(&g),
				Promise:    controller.NewPromiseController(&g),
				Health:     controller.NewHealthController(&g),
				Metrics:    controller.NewMetricsController(&g),
			})

			t.Run("topics", func(t *testing.T) {
				t.Parallel()

				t.Run("get", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodGet, "/topics", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"data":[`)
					r.AssertBodyContains(`{"name":"topic"}`)
				})

				t.Run("delete", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodDelete, "/topics", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"deleted":`)
				})
			})

			t.Run("topic", func(t *testing.T) {
				t.Parallel()

				t.Run("get", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodGet, "/topics/topic", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"name":"topic"`)
					r.AssertBodyContains(`"count":`)
				})

				t.Run("delete", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodDelete, "/topics/topic", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"deleted":`)
				})
			})

			t.Run("tasks", func(t *testing.T) {
				t.Parallel()

				t.Run("get", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodGet, "/topics/topic/tasks", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"data":[`)
					r.AssertBodyContains(`"topic":"topic`)
				})

				t.Run("post", func(t *testing.T) {
					t.Parallel()
					v := ratus.Tasks{Data: []*ratus.Task{{ID: "id"}}}
					req := reqtest.NewRequestJSON(http.MethodPost, "/topics/topic/tasks", &v)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusCreated)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"created":`)
					r.AssertBodyContains(`"updated":`)
				})

				t.Run("put", func(t *testing.T) {
					t.Parallel()
					v := ratus.Tasks{Data: []*ratus.Task{{ID: "id"}}}
					req := reqtest.NewRequestJSON(http.MethodPut, "/topics/topic/tasks", &v)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusCreated)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"created":`)
					r.AssertBodyContains(`"updated":`)
				})

				t.Run("delete", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodDelete, "/topics/topic/tasks", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"deleted":`)
				})
			})

			t.Run("task", func(t *testing.T) {
				t.Parallel()

				t.Run("get", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodGet, "/topics/topic/tasks/id", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"topic":"topic`)
				})

				t.Run("post", func(t *testing.T) {
					t.Parallel()
					var v ratus.Task
					req := reqtest.NewRequestJSON(http.MethodPost, "/topics/topic/tasks/id", &v)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusCreated)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"created":`)
					r.AssertBodyContains(`"updated":`)
				})

				t.Run("put", func(t *testing.T) {
					t.Parallel()
					var v ratus.Task
					req := reqtest.NewRequestJSON(http.MethodPut, "/topics/topic/tasks/id", &v)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"created":`)
					r.AssertBodyContains(`"updated":`)
				})

				t.Run("delete", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodDelete, "/topics/topic/tasks/id", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"deleted":`)
				})

				t.Run("patch", func(t *testing.T) {
					t.Parallel()
					v := ratus.Commit{Topic: "topic"}
					req := reqtest.NewRequestJSON(http.MethodPatch, "/topics/topic/tasks/id", &v)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"topic":"topic`)
				})
			})

			t.Run("promises", func(t *testing.T) {
				t.Parallel()

				t.Run("get", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodGet, "/topics/topic/promises", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"data":[`)
					r.AssertBodyContains(`"deadline":`)
				})

				t.Run("post", func(t *testing.T) {
					t.Parallel()
					var v ratus.Promise
					req := reqtest.NewRequestJSON(http.MethodPost, "/topics/topic/promises", &v)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"topic":"topic`)
				})

				t.Run("delete", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodDelete, "/topics/topic/promises", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"deleted":`)
				})

				t.Run("redirect", func(t *testing.T) {
					t.Parallel()
					v := ratus.Promise{ID: "id"}
					req := reqtest.NewRequestJSON(http.MethodPost, "/topics/topic/promises", &v)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"topic":"topic`)
				})
			})

			t.Run("promise", func(t *testing.T) {
				t.Parallel()

				t.Run("get", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodGet, "/topics/topic/promises/id", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"deadline":`)
				})

				t.Run("post", func(t *testing.T) {
					t.Parallel()
					var v ratus.Promise
					req := reqtest.NewRequestJSON(http.MethodPost, "/topics/topic/promises/id", &v)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"topic":"topic`)
				})

				t.Run("put", func(t *testing.T) {
					t.Parallel()
					var v ratus.Promise
					req := reqtest.NewRequestJSON(http.MethodPut, "/topics/topic/promises/id", &v)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"topic":"topic`)
				})

				t.Run("delete", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodDelete, "/topics/topic/promises/id", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains(`"deleted":`)
				})
			})

			t.Run("health", func(t *testing.T) {
				t.Parallel()

				t.Run("livez", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodGet, "/livez", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
				})

				t.Run("readyz", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
				})
			})

			t.Run("metrics", func(t *testing.T) {
				t.Parallel()

				t.Run("get", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusOK)
				})
			})
		})

		t.Run("conflict", func(t *testing.T) {
			t.Parallel()
			o := config.PaginationConfig{MaxLimit: 10, MaxOffset: 10}
			g := stub.Engine{Err: ratus.ErrConflict}
			h := reqtest.NewHandler(&controller.V1{
				Pagination: middleware.Pagination(&o),
				Topic:      controller.NewTopicController(&g),
				Task:       controller.NewTaskController(&g),
				Promise:    controller.NewPromiseController(&g),
				Health:     controller.NewHealthController(&g),
				Metrics:    controller.NewMetricsController(&g),
			})

			t.Run("task", func(t *testing.T) {
				t.Parallel()

				t.Run("post", func(t *testing.T) {
					t.Parallel()
					var v ratus.Task
					req := reqtest.NewRequestJSON(http.MethodPost, "/topics/topic/tasks/id", &v)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusConflict)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains("a task with the same ID already exists")
				})

				t.Run("patch", func(t *testing.T) {
					t.Parallel()
					var v ratus.Commit
					req := reqtest.NewRequestJSON(http.MethodPatch, "/topics/topic/tasks/id", &v)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusConflict)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains("the task may have been modified by others")
				})
			})

			t.Run("promise", func(t *testing.T) {
				t.Parallel()

				t.Run("post", func(t *testing.T) {
					t.Parallel()
					var v ratus.Promise
					req := reqtest.NewRequestJSON(http.MethodPost, "/topics/topic/promises/id", &v)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusConflict)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains("the target task is not in pending state")
				})
			})
		})

		t.Run("unavailable", func(t *testing.T) {
			t.Parallel()
			o := config.PaginationConfig{MaxLimit: 10, MaxOffset: 10}
			g := stub.Engine{Err: ratus.ErrServiceUnavailable}
			h := reqtest.NewHandler(&controller.V1{
				Pagination: middleware.Pagination(&o),
				Topic:      controller.NewTopicController(&g),
				Task:       controller.NewTaskController(&g),
				Promise:    controller.NewPromiseController(&g),
				Health:     controller.NewHealthController(&g),
				Metrics:    controller.NewMetricsController(&g),
			})

			t.Run("health", func(t *testing.T) {
				t.Parallel()

				t.Run("readyz", func(t *testing.T) {
					t.Parallel()
					req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
					r := reqtest.Record(t, h, req)
					r.AssertStatusCode(http.StatusServiceUnavailable)
					r.AssertHeaderContains("Content-Type", "application/json")
					r.AssertBodyContains("unavailable")
				})
			})
		})
	})
}
