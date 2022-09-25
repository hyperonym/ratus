package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/config"
	"github.com/hyperonym/ratus/internal/middleware"
	"github.com/hyperonym/ratus/internal/reqtest"
)

type group struct{}

func (g *group) Prefixes() []string {
	return []string{"/"}
}

func (g *group) Mount(r *gin.RouterGroup) {
	p := promhttp.Handler()

	r.GET("/prometheus", middleware.Prometheus(), func(c *gin.Context) {
		p.ServeHTTP(c.Writer, c.Request)
	})

	r.GET("/pagination/20", middleware.Pagination(&config.PaginationConfig{
		MaxLimit:  20,
		MaxOffset: 20,
	}), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"limit":  c.GetInt(middleware.ParamLimit),
			"offset": c.GetInt(middleware.ParamOffset),
		})
	})

	r.GET("/pagination/5", middleware.Pagination(&config.PaginationConfig{
		MaxLimit:  5,
		MaxOffset: 5,
	}), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"limit":  c.GetInt(middleware.ParamLimit),
			"offset": c.GetInt(middleware.ParamOffset),
		})
	})

	r.POST("/topics/:topic/tasks/:id", middleware.Task(), func(c *gin.Context) {
		c.JSON(http.StatusOK, c.MustGet(middleware.ParamTask))
	})

	r.POST("/topics/:topic/tasks", middleware.Tasks(), func(c *gin.Context) {
		c.JSON(http.StatusOK, c.MustGet(middleware.ParamTasks))
	})

	r.POST("/topics/:topic/promises/:id", middleware.Promise(), func(c *gin.Context) {
		c.JSON(http.StatusOK, c.MustGet(middleware.ParamPromise))
	})

	r.PATCH("/topics/:topic/tasks/:id", middleware.Commit(), func(c *gin.Context) {
		c.JSON(http.StatusOK, c.MustGet(middleware.ParamCommit))
	})
}

func TestMiddleware(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	h := reqtest.NewHandler(&group{})

	t.Run("prometheus", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/prometheus", nil)
		r1 := reqtest.Record(t, h, req)
		r1.AssertStatusCode(http.StatusOK)
		r2 := reqtest.Record(t, h, req)
		r2.AssertStatusCode(http.StatusOK)
		r2.AssertBodyContains("ratus_request_duration_seconds_bucket")
		r2.AssertBodyContains(`endpoint="/prometheus"`)
	})

	t.Run("pagination", func(t *testing.T) {
		t.Parallel()

		t.Run("20", func(t *testing.T) {
			t.Parallel()

			t.Run("default", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/pagination/20?offset=5", nil)
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusOK)
				r.AssertBodyContains(`"limit":10`)
				r.AssertBodyContains(`"offset":5`)
			})

			t.Run("bind", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/pagination/20?offset=foo", nil)
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusBadRequest)
				r.AssertBodyContains("invalid pagination parameters")
			})

			t.Run("limit", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/pagination/20?limit=21", nil)
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusBadRequest)
				r.AssertBodyContains("exceeded maximum allowed limit of 20")
			})

			t.Run("offset", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/pagination/20?offset=21", nil)
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusBadRequest)
				r.AssertBodyContains("exceeded maximum allowed offset of 20")
			})
		})

		t.Run("5", func(t *testing.T) {
			t.Parallel()

			t.Run("default", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/pagination/5", nil)
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusOK)
				r.AssertBodyContains(`"limit":5`)
				r.AssertBodyContains(`"offset":0`)
			})

			t.Run("limit", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/pagination/5?limit=-1", nil)
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusBadRequest)
				r.AssertBodyContains("limit must not be negative")
			})

			t.Run("offset", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/pagination/5?offset=-1", nil)
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusBadRequest)
				r.AssertBodyContains("offset must not be negative")
			})
		})
	})

	t.Run("task", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()
			req := reqtest.NewRequestJSON(http.MethodPost, "/topics/test/tasks/1", &ratus.Task{})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusOK)
			r.AssertHeaderContains("Content-Type", "application/json")
			r.AssertBodyContains(`"_id":"1"`)
			r.AssertBodyContains(`"topic":"test"`)
			r.AssertBodyContains(`"state":0`)
			r.AssertBodyContains(`"produced":"`)
			r.AssertBodyContains(`"scheduled":`)
		})

		t.Run("bind", func(t *testing.T) {
			t.Parallel()
			req := reqtest.NewRequestJSON(http.MethodPost, "/topics/test/tasks/1", gin.H{"_id": 1})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusBadRequest)
			r.AssertBodyContains("cannot unmarshal number")
		})

		t.Run("eof", func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/topics/test/tasks/1", nil)
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusBadRequest)
			r.AssertBodyContains("missing request body")
		})

		t.Run("id", func(t *testing.T) {
			t.Parallel()
			req := reqtest.NewRequestJSON(http.MethodPost, "/topics/test/tasks/1", &ratus.Task{ID: "2"})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusBadRequest)
			r.AssertBodyContains("inconsistent with the path parameter")
		})

		t.Run("topic", func(t *testing.T) {
			t.Parallel()
			req := reqtest.NewRequestJSON(http.MethodPost, "/topics//tasks/1", &ratus.Task{})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusBadRequest)
			r.AssertBodyContains("topic must not be empty")
		})

		t.Run("state", func(t *testing.T) {
			t.Parallel()
			req := reqtest.NewRequestJSON(http.MethodPost, "/topics/test/tasks/1", &ratus.Task{State: 9})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusBadRequest)
			r.AssertBodyContains("invalid state")
		})

		t.Run("defer", func(t *testing.T) {
			t.Parallel()

			t.Run("normal", func(t *testing.T) {
				t.Parallel()
				req := reqtest.NewRequestJSON(http.MethodPost, "/topics/test/tasks/1", &ratus.Task{Defer: "10m"})
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusOK)
			})

			t.Run("invalid", func(t *testing.T) {
				t.Parallel()
				req := reqtest.NewRequestJSON(http.MethodPost, "/topics/test/tasks/1", &ratus.Task{Defer: "foo"})
				r := reqtest.Record(t, h, req)
				r.AssertStatusCode(http.StatusBadRequest)
				r.AssertBodyContains("invalid duration")
			})
		})
	})

	t.Run("tasks", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()
			req := reqtest.NewRequestJSON(http.MethodPost, "/topics/test/tasks", &ratus.Tasks{
				Data: []*ratus.Task{{ID: "1"}},
			})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusOK)
			r.AssertHeaderContains("Content-Type", "application/json")
			r.AssertBodyContains(`"_id":"1"`)
			r.AssertBodyContains(`"topic":"test"`)
			r.AssertBodyContains(`"data":[`)
		})

		t.Run("bind", func(t *testing.T) {
			t.Parallel()
			req := reqtest.NewRequestJSON(http.MethodPost, "/topics/test/tasks", gin.H{"data": 1})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusBadRequest)
			r.AssertBodyContains("cannot unmarshal number")
		})

		t.Run("eof", func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/topics/test/tasks", nil)
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusBadRequest)
			r.AssertBodyContains("missing request body")
		})

		t.Run("empty", func(t *testing.T) {
			t.Parallel()
			req := reqtest.NewRequestJSON(http.MethodPost, "/topics/test/tasks", &ratus.Tasks{})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusOK)
			r.AssertBodyContains(`{"data":[]}`)
		})

		t.Run("id", func(t *testing.T) {
			t.Parallel()
			req := reqtest.NewRequestJSON(http.MethodPost, "/topics/test/tasks", &ratus.Tasks{
				Data: []*ratus.Task{{Producer: "foo"}},
			})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusBadRequest)
			r.AssertBodyContains("ID must not be empty")
		})
	})

	t.Run("promise", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()
			req := reqtest.NewRequestJSON(http.MethodPost, "/topics/test/promises/1?consumer=foo", &ratus.Promise{})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusOK)
			r.AssertHeaderContains("Content-Type", "application/json")
			r.AssertBodyContains(`"_id":"1"`)
			r.AssertBodyContains(`"consumer":"foo"`)
			r.AssertBodyContains(`"deadline":`)
		})

		t.Run("id", func(t *testing.T) {
			t.Parallel()
			req := reqtest.NewRequestJSON(http.MethodPost, "/topics/test/promises/1", &ratus.Promise{ID: "2"})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusBadRequest)
			r.AssertBodyContains("inconsistent with the path parameter")
		})

		t.Run("timeout", func(t *testing.T) {
			t.Parallel()
			req := reqtest.NewRequestJSON(http.MethodPost, "/topics/test/promises/1", &ratus.Promise{Timeout: "foo"})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusBadRequest)
			r.AssertBodyContains("invalid duration")
		})
	})

	t.Run("commit", func(t *testing.T) {
		t.Parallel()

		t.Run("normal", func(t *testing.T) {
			t.Parallel()
			req := reqtest.NewRequestJSON(http.MethodPatch, "/topics/test/tasks/1", &ratus.Commit{Defer: "10m"})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusOK)
			r.AssertHeaderContains("Content-Type", "application/json")
			r.AssertBodyContains(`"state":2`)
			r.AssertBodyContains(`"scheduled":`)
		})

		t.Run("state", func(t *testing.T) {
			t.Parallel()
			var s ratus.TaskState = 9
			req := reqtest.NewRequestJSON(http.MethodPatch, "/topics/test/tasks/1", &ratus.Commit{State: &s})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusBadRequest)
			r.AssertBodyContains("invalid target state")
		})

		t.Run("defer", func(t *testing.T) {
			t.Parallel()
			req := reqtest.NewRequestJSON(http.MethodPatch, "/topics/test/tasks/1", &ratus.Commit{Defer: "foo"})
			r := reqtest.Record(t, h, req)
			r.AssertStatusCode(http.StatusBadRequest)
			r.AssertBodyContains("invalid duration")
		})
	})
}
