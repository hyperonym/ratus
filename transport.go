package ratus

import (
	"net/http"
	"net/url"
)

// maxIdleConnsPerHost is the maximum number of idle (keep-alive) connections.
const maxIdleConnsPerHost = 1024

// transport wraps around http.DefaultTransport to rewrite origins and set HTTP
// headers for all outgoing requests.
type transport struct {
	scheme       string
	host         string
	headers      map[string]string
	roundTripper http.RoundTripper
}

// newTransport creates a custom transport instance.
func newTransport(origin string, headers map[string]string) (*transport, error) {

	// Parse the origin string to extract URL components for rewriting.
	u, err := url.Parse(origin)
	if err != nil {
		return nil, err
	}

	// Inherit settings from http.DefaultTransport by cloning it.
	t := http.DefaultTransport.(*http.Transport).Clone()

	// Remove limits of maximum number of connections.
	t.MaxIdleConns = 0
	t.MaxConnsPerHost = 0
	t.MaxIdleConnsPerHost = maxIdleConnsPerHost

	return &transport{
		scheme:       u.Scheme,
		host:         u.Host,
		headers:      headers,
		roundTripper: t,
	}, nil
}

// RoundTrip implements the http.RoundTripper interface.
func (t *transport) RoundTrip(r *http.Request) (*http.Response, error) {

	// Rewrite request URL components.
	if t.scheme != "" {
		r.URL.Scheme = t.scheme
	}
	if t.host != "" {
		r.URL.Host = t.host
	}

	// Set common header fields.
	for k, v := range t.headers {
		if _, ok := r.Header[k]; !ok {
			r.Header.Set(k, v)
		}
	}

	// Set User-Agent if it is not present.
	if _, ok := r.Header["User-Agent"]; !ok {
		r.Header.Set("User-Agent", "Ratus-Client")
	}

	// Execute the modified HTTP transaction.
	return t.roundTripper.RoundTrip(r)
}
