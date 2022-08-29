package reqtest_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hyperonym/ratus/internal/reqtest"
)

func TestStub(t *testing.T) {
	h := reqtest.NewHandler(&reqtest.StubGroup{})
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	r := reqtest.Record(t, h, req)
	r.AssertStatusCode(http.StatusOK)
	r.AssertHeaderContains("Content-Type", "text/plain")
	r.AssertBodyContains("42")
}
