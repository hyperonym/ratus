// Package reqtest provides utilities for testing requests and responses.
package reqtest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ResponseRecord contains the status code, header, and body of the recorded
// response. More fields may be populated in the future, so callers should not
// DeepEqual the result in tests. Use the provided assertion methods instead.
type ResponseRecord struct {
	StatusCode int
	Header     http.Header
	Body       []byte

	// Keep a reference to the test/subtest for assertion methods.
	test *testing.T
}

// AssertStatusCode marks the test as failed if the response status code does
// not match the expected value.
func (r *ResponseRecord) AssertStatusCode(v int) {
	r.test.Helper()
	if r.StatusCode != v {
		r.test.Errorf("incorrect response status code, expected %d, got %d", v, r.StatusCode)
	}
}

// AssertHeaderContains marks the test as failed if the response header field
// does not contain the provided substring.
func (r *ResponseRecord) AssertHeaderContains(field string, v string) {
	r.test.Helper()
	if h := r.Header.Get(field); !strings.Contains(h, v) {
		r.test.Errorf("response header field %q does not contain %q, got %q", field, v, h)
	}
}

// AssertBodyContains marks the test as failed if the response body does not
// contain the provided substring.
func (r *ResponseRecord) AssertBodyContains(v string) {
	r.test.Helper()
	if !bytes.Contains(r.Body, []byte(v)) {
		r.test.Errorf("response body does not contain %q, got %q", v, string(r.Body))
	}
}

// Record the response generated by the handler for testing.
func Record(t *testing.T, h http.Handler, req *http.Request) *ResponseRecord {
	t.Helper()

	// Handle the request and write the response to a recorder.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	// Read and save the response body as bytes.
	d, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	return &ResponseRecord{
		StatusCode: res.StatusCode,
		Header:     res.Header,
		Body:       d,
		test:       t,
	}
}

// NewRequestJSON wraps around httptest.NewRequest to return a new incoming
// server Request containing a JSON encoded request body. NewRequestJSON
// panics on error for ease of use in testing, where a panic is acceptable.
func NewRequestJSON(method, target string, body any) *http.Request {

	// Encode the request body in JSON.
	var b io.Reader
	if body != nil {
		d, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		b = bytes.NewBuffer(d)
	}

	// Create request and set the content type header.
	req := httptest.NewRequest(method, target, b)
	req.Header.Set("Content-Type", "application/json")

	return req
}
