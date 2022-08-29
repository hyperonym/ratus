// Package config contains configurations and command line arguments.
package config

import "time"

// ServerConfig contains configurations for the API server.
type ServerConfig struct {
	Port uint   `arg:"-p,--port,env:PORT" placeholder:"PORT" help:"port on which to listen for API requests" default:"80"`
	Bind string `arg:"-b,--bind,env:BIND" placeholder:"ADDR" help:"address on which to listen for API requests" default:"0.0.0.0"`
}

// ChoreConfig contains configurations for background jobs.
type ChoreConfig struct {
	Interval      time.Duration `arg:"--chore-interval,env:CHORE_INTERVAL" placeholder:"DURATION" help:"interval for running periodic background jobs such as recovering and expiring tasks" default:"10s"`
	InitialDelay  time.Duration `arg:"--chore-initial-delay,env:CHORE_INITIAL_DELAY" placeholder:"DURATION" help:"delay before the initial execution of background jobs to avoid spikes while starting multiple instances" default:"0s"`
	InitialRandom bool          `arg:"--chore-initial-random,env:CHORE_INITIAL_RANDOM" help:"randomly defer the initial execution of background jobs within a range that does not exceed the initial delay"`
}

// PaginationConfig contains configurations for pagination.
type PaginationConfig struct {
	MaxLimit  int `arg:"--pagination-max-limit,env:PAGINATION_MAX_LIMIT" placeholder:"LIMIT" help:"maximum number of resources to return in pagination" default:"100"`
	MaxOffset int `arg:"--pagination-max-offset,env:PAGINATION_MAX_OFFSET" placeholder:"OFFSET" help:"maximum number of resources to be skipped in pagination" default:"10000"`
}
