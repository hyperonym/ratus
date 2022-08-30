package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hyperonym/ratus/internal/metrics"
	"github.com/hyperonym/ratus/internal/reqtest"
)

func TestMetrics(t *testing.T) {
	h := promhttp.Handler()

	metrics.RequestHistogram.WithLabelValues("test", "/foo", "404").Observe(0.42)
	metrics.ChoreHistogram.Observe(0.42)
	metrics.DelayGauge.WithLabelValues("test", "foo", "bar").Set(42)
	metrics.ExecutionGauge.WithLabelValues("test", "foo", "bar").Set(42)
	metrics.ProducedCounter.WithLabelValues("test", "foo").Add(42)
	metrics.ConsumedCounter.WithLabelValues("test", "foo", "bar").Add(42)
	metrics.CommittedCounter.WithLabelValues("test", "foo", "bar").Add(42)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	r := reqtest.Record(t, h, req)
	r.AssertStatusCode(http.StatusOK)
	r.AssertBodyContains("ratus_request_duration_seconds")
	r.AssertBodyContains(`ratus_chore_duration_seconds_bucket{le="0.1"} 0`)
	r.AssertBodyContains(`ratus_chore_duration_seconds_bucket{le="0.5"} 1`)
	r.AssertBodyContains("ratus_task_schedule_delay_seconds")
	r.AssertBodyContains("ratus_task_execution_duration_seconds")
	r.AssertBodyContains("ratus_task_produced_count_total")
	r.AssertBodyContains("ratus_task_consumed_count_total")
	r.AssertBodyContains("ratus_task_committed_count_total")
	r.AssertBodyContains(`topic="test"`)
	r.AssertBodyContains(`producer="foo"`)
	r.AssertBodyContains(`consumer="bar"`)
	r.AssertBodyContains("} 42")
}
