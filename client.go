package ratus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/sync/errgroup"
)

// DefaultConcurrencyDelay is the default value of SubscribeOptions's ConcurrencyDelay.
const DefaultConcurrencyDelay = 1 * time.Second

// DefaultDrainInterval is the default value of SubscribeOptions's DrainInterval.
const DefaultDrainInterval = 5 * time.Second

// DefaultErrorInterval is the default value of SubscribeOptions's ErrorInterval.
const DefaultErrorInterval = 30 * time.Second

// ClientOptions contains options to configure a Ratus client.
type ClientOptions struct {

	// Origin of the Ratus instance or load balancer to connect to.
	// An origin is a combination of a scheme, hostname, and port.
	// Reference: https://web.dev/same-site-same-origin/#origin
	Origin string

	// Common header key-value pairs for every outgoing request.
	Headers map[string]string

	// Timeout specifies a time limit for requests made by this client.
	// This is not related to the timeout for task execution.
	// A Timeout of zero means no timeout.
	Timeout time.Duration
}

// Client is an HTTP client that talks to Ratus.
type Client struct {
	client *http.Client
}

// NewClient creates a new Ratus client instance.
func NewClient(o *ClientOptions) (*Client, error) {

	// Create a custom transport to rewrite all requests to the specified
	// origin, allowing users to call API endpoints using relative paths.
	t, err := newTransport(o.Origin, o.Headers)
	if err != nil {
		return nil, err
	}

	// Create the internal HTTP client using the custom transport.
	c := http.Client{
		Transport: t,
		Timeout:   o.Timeout,
	}

	return &Client{&c}, nil
}

// SubscribeOptions contains options for subscribing to a topic.
type SubscribeOptions struct {

	// A wildcard promise containing consumer and timeout settings.
	// The promise will be reused for all polling operations, thus the ID and
	// deadline fields will be ignored.
	Promise *Promise
	// Name of the topic to subscribe to.
	Topic string

	// Maximum number of tasks to be executed concurrently.
	Concurrency int
	// Delay added before starting each polling goroutine to avoid spikes.
	// If zero, DefaultConcurrencyDelay is used.
	ConcurrencyDelay time.Duration

	// Pause duration after successful polls.
	// By default will proceed to the next poll immediately without pausing.
	PollInterval time.Duration
	// Pause duration when the topic has been emptied.
	// If zero, DefaultDrainInterval is used.
	DrainInterval time.Duration
	// Pause duration when an error occurs.
	// If zero, DefaultErrorInterval is used.
	ErrorInterval time.Duration
}

// SubscribeHandler defines the signature of handler functions for the
// Subscribe method.
type SubscribeHandler func(ctx *Context, err error)

// Subscribe to a topic and attach a handler function to listen for new tasks
// and errors. Calling Subscribe will block the calling goroutine indefinitely
// unless the context times out or gets canceled.
func (c *Client) Subscribe(ctx context.Context, o *SubscribeOptions, f SubscribeHandler) error {

	// Copy consumer and timeout settings to create a reusable wildcard promise.
	p := &Promise{
		Consumer: o.Promise.Consumer,
		Timeout:  o.Promise.Timeout,
	}

	// Check the options and use the default values if required.
	n := o.Concurrency
	if n < 1 {
		n = 1
	}
	cd := o.ConcurrencyDelay
	if cd <= 0 {
		cd = DefaultConcurrencyDelay
	}
	dd := o.DrainInterval
	if dd <= 0 {
		dd = DefaultDrainInterval
	}
	ed := o.ErrorInterval
	if ed <= 0 {
		ed = DefaultErrorInterval
	}

	// Start polling goroutines with a delay between each two to avoid spikes.
	e, ctx := errgroup.WithContext(ctx)
	for i := 0; i < n; i++ {
		d := cd * time.Duration(i)
		e.Go(func() error {
			r := time.NewTimer(d)
			xc := make(chan *Context, 1)
			ec := make(chan error, 1)
			for {
				select {
				case <-ctx.Done():
					r.Stop()
					return ctx.Err()
				case <-r.C:
					x, err := c.Poll(ctx, o.Topic, p)
					if err != nil {
						ec <- err
						break
					}
					xc <- x
				case x := <-xc:
					f(x, nil)

					// Automatically commit the updates if no commit has been
					// made explicitly in the handler function.
					if err := x.Commit(); err != nil {
						ec <- err
						break
					}
					r.Reset(o.PollInterval)
				case err := <-ec:

					// The topic has been emptied or no task has reached its
					// scheduled time of execution, then poll again later.
					if errors.Is(err, ErrNotFound) {
						r.Reset(dd)
						break
					}

					// Handle unexpected errors.
					if ctx.Err() == nil {
						f(nil, err)
						r.Reset(ed)
					}
				}
			}
		})
	}

	return e.Wait()
}

// Poll claims and returns the next available task in a topic.
// An error wrapping ErrNotFound is returned if the topic is empty,
// or if no task in the topic has reached its scheduled time of execution.
func (c *Client) Poll(ctx context.Context, topic string, p *Promise) (*Context, error) {

	// Get the next available task in the topic.
	t, err := c.PostPromises(ctx, topic, p)
	if err != nil {
		return nil, err
	}

	// Create context with a timeout calculated from the deadline of the task.
	// To avoid clock synchronization issues, instead of using deadline directly,
	// use the time difference between the task's deadline and consumed time as
	// the timeout duration for the context.
	var n context.CancelFunc
	if t.Consumed != nil && t.Deadline != nil {
		ctx, n = context.WithTimeout(ctx, t.Deadline.Sub(*t.Consumed))
	}

	// Create commit instance with the default target state set as "completed".
	// The nonce string from the task is also populated to enable strict mode.
	s := TaskStateCompleted
	m := Commit{
		Nonce: t.Nonce,
		State: &s,
	}

	return &Context{
		Context: ctx,
		cancel:  n,
		commit:  m,
		client:  c,
		Task:    t,
	}, nil
}

// Request calls an API endpoint and stores the response body in the value
// pointed to by result. Error messages from Ratus will be translated into
// errors and returned.
func (c *Client) Request(ctx context.Context, method, endpoint string, body, result any) error {

	// Encode the request body in JSON.
	var b io.Reader
	if body != nil {
		d, err := json.Marshal(body)
		if err != nil {
			return err
		}
		b = bytes.NewBuffer(d)
	}

	// Create request and execute it using the internal HTTP client.
	req, err := http.NewRequestWithContext(ctx, method, endpoint, b)
	if err != nil {
		return err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Handle failed request and parse the error message.
	if res.StatusCode >= http.StatusBadRequest {
		var r Error
		if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
			return err
		}
		return r.Err()
	}

	// Discard the response body if unmarshalling is not required.
	if result == nil {
		io.Copy(io.Discard, res.Body)
		return err
	}

	return json.NewDecoder(res.Body).Decode(result)
}

// ListTopics lists all topics.
func (c *Client) ListTopics(ctx context.Context, limit, offset int) ([]*Topic, error) {
	var v Topics
	if err := c.Request(ctx, http.MethodGet, fmt.Sprintf("/v1/topics?limit=%d&offset=%d", limit, offset), nil, &v); err != nil {
		return nil, err
	}
	return v.Data, nil
}

// DeleteTopics deletes all topics and tasks.
func (c *Client) DeleteTopics(ctx context.Context) (*Deleted, error) {
	var v Deleted
	if err := c.Request(ctx, http.MethodDelete, "/v1/topics", nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// GetTopic gets information about a topic.
func (c *Client) GetTopic(ctx context.Context, topic string) (*Topic, error) {
	var v Topic
	if err := c.Request(ctx, http.MethodGet, fmt.Sprintf("/v1/topics/%s", url.PathEscape(topic)), nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// DeleteTopic deletes a topic and its tasks.
func (c *Client) DeleteTopic(ctx context.Context, topic string) (*Deleted, error) {
	var v Deleted
	if err := c.Request(ctx, http.MethodDelete, fmt.Sprintf("/v1/topics/%s", url.PathEscape(topic)), nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// ListTasks lists all tasks in a topic.
func (c *Client) ListTasks(ctx context.Context, topic string, limit, offset int) ([]*Task, error) {
	var v Tasks
	if err := c.Request(ctx, http.MethodGet, fmt.Sprintf("/v1/topics/%s/tasks?limit=%d&offset=%d", url.PathEscape(topic), limit, offset), nil, &v); err != nil {
		return nil, err
	}
	return v.Data, nil
}

// InsertTasks inserts a batch of tasks while ignoring existing ones.
func (c *Client) InsertTasks(ctx context.Context, ts []*Task) (*Updated, error) {
	var v Updated
	if err := c.Request(ctx, http.MethodPost, "/v1/topics//tasks", &Tasks{ts}, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// UpsertTasks inserts or updates a batch of tasks.
func (c *Client) UpsertTasks(ctx context.Context, ts []*Task) (*Updated, error) {
	var v Updated
	if err := c.Request(ctx, http.MethodPut, "/v1/topics//tasks", &Tasks{ts}, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// DeleteTasks deletes all tasks in a topic.
func (c *Client) DeleteTasks(ctx context.Context, topic string) (*Deleted, error) {
	var v Deleted
	if err := c.Request(ctx, http.MethodDelete, fmt.Sprintf("/v1/topics/%s/tasks", url.PathEscape(topic)), nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// GetTask gets a task by its unique ID.
func (c *Client) GetTask(ctx context.Context, id string) (*Task, error) {
	var v Task
	if err := c.Request(ctx, http.MethodGet, fmt.Sprintf("/v1/topics//tasks/%s", url.PathEscape(id)), nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// InsertTask inserts a new task.
func (c *Client) InsertTask(ctx context.Context, t *Task) (*Updated, error) {
	var v Updated
	if err := c.Request(ctx, http.MethodPost, fmt.Sprintf("/v1/topics//tasks/%s", url.PathEscape(t.ID)), t, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// UpsertTask inserts or updates a task.
func (c *Client) UpsertTask(ctx context.Context, t *Task) (*Updated, error) {
	var v Updated
	if err := c.Request(ctx, http.MethodPut, fmt.Sprintf("/v1/topics//tasks/%s", url.PathEscape(t.ID)), t, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// DeleteTask deletes a task by its unique ID.
func (c *Client) DeleteTask(ctx context.Context, id string) (*Deleted, error) {
	var v Deleted
	if err := c.Request(ctx, http.MethodDelete, fmt.Sprintf("/v1/topics//tasks/%s", url.PathEscape(id)), nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// PatchTask applies a set of updates to a task and returns the updated task.
func (c *Client) PatchTask(ctx context.Context, id string, m *Commit) (*Task, error) {
	var v Task
	if err := c.Request(ctx, http.MethodPatch, fmt.Sprintf("/v1/topics//tasks/%s", url.PathEscape(id)), m, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// ListPromises lists all promises in a topic.
func (c *Client) ListPromises(ctx context.Context, topic string, limit, offset int) ([]*Promise, error) {
	var v Promises
	if err := c.Request(ctx, http.MethodGet, fmt.Sprintf("/v1/topics/%s/promises?limit=%d&offset=%d", url.PathEscape(topic), limit, offset), nil, &v); err != nil {
		return nil, err
	}
	return v.Data, nil
}

// PostPromises makes a promise to claim and execute the next available task in a topic.
func (c *Client) PostPromises(ctx context.Context, topic string, p *Promise) (*Task, error) {
	var v Task
	if err := c.Request(ctx, http.MethodPost, fmt.Sprintf("/v1/topics/%s/promises", url.PathEscape(topic)), p, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// DeletePromises deletes all promises in a topic.
func (c *Client) DeletePromises(ctx context.Context, topic string) (*Deleted, error) {
	var v Deleted
	if err := c.Request(ctx, http.MethodDelete, fmt.Sprintf("/v1/topics/%s/promises", url.PathEscape(topic)), nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// GetPromise gets a promise by the unique ID of its target task.
func (c *Client) GetPromise(ctx context.Context, id string) (*Promise, error) {
	var v Promise
	if err := c.Request(ctx, http.MethodGet, fmt.Sprintf("/v1/topics//promises/%s", url.PathEscape(id)), nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// InsertPromise makes a promise to claim and execute a task if it is in pending state.
func (c *Client) InsertPromise(ctx context.Context, p *Promise) (*Task, error) {
	var v Task
	if err := c.Request(ctx, http.MethodPost, fmt.Sprintf("/v1/topics//promises/%s", url.PathEscape(p.ID)), p, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// UpsertPromise makes a promise to claim and execute a task regardless of its current state.
func (c *Client) UpsertPromise(ctx context.Context, p *Promise) (*Task, error) {
	var v Task
	if err := c.Request(ctx, http.MethodPut, fmt.Sprintf("/v1/topics//promises/%s", url.PathEscape(p.ID)), p, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// DeletePromise deletes a promise by the unique ID of its target task.
func (c *Client) DeletePromise(ctx context.Context, id string) (*Deleted, error) {
	var v Deleted
	if err := c.Request(ctx, http.MethodDelete, fmt.Sprintf("/v1/topics//promises/%s", url.PathEscape(id)), nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// GetLiveness checks the liveness of the instance.
func (c *Client) GetLiveness(ctx context.Context) error {
	return c.Request(ctx, http.MethodGet, "/v1/livez", nil, nil)
}

// GetReadiness checks the readiness of the instance.
func (c *Client) GetReadiness(ctx context.Context) error {
	return c.Request(ctx, http.MethodGet, "/v1/readyz", nil, nil)
}
