// Package memdb implements the storage engine interface for MemDB.
package memdb

import (
	"context"
	"time"

	"github.com/hashicorp/go-memdb"

	"github.com/hyperonym/ratus"
)

// Name constants for tables.
const tableTask = "task"

// Name constants for fields.
const (
	keyID        = "ID"
	keyTopic     = "Topic"
	keyState     = "State"
	keyScheduled = "Scheduled"
	keyConsumed  = "Consumed"
	keyDeadline  = "Deadline"
)

// Name constants for index creation and selection.
const (
	indexID                    = "id"
	indexTopic                 = "topic"
	indexPendingTopicScheduled = "pending-topic-scheduled"
	indexActiveDeadline        = "active-deadline"
	indexActiveTopic           = "active-topic"
	indexCompletedConsumed     = "completed-consumed "
)

// Config contains configurations for the MemDB storage engine.
type Config struct {
	RetentionPeriod time.Duration `arg:"--memdb-retention-period,env:MEMDB_RETENTION_PERIOD" placeholder:"DURATION" help:"retention period for completed tasks" default:"72h"`
}

// Engine implements the storage engine interface for MemDB.
type Engine struct {
	config   *Config
	schema   *memdb.DBSchema
	database *memdb.MemDB
}

// New creates a new MemDB storage engine instance.
func New(c *Config) (*Engine, error) {

	// Create the database schema.
	s := memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			tableTask: {
				Name: tableTask,
				Indexes: map[string]*memdb.IndexSchema{
					indexID: {
						Name:         indexID,
						AllowMissing: false,
						Unique:       true,
						Indexer:      &memdb.StringFieldIndex{Field: keyID},
					},
					indexTopic: {
						Name:         indexTopic,
						AllowMissing: false,
						Unique:       false,
						Indexer:      &memdb.StringFieldIndex{Field: keyTopic},
					},
					indexPendingTopicScheduled: {
						Name:         indexPendingTopicScheduled,
						AllowMissing: true,
						Unique:       false,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&StateFieldIndex{Field: keyState, Filter: ratus.TaskStatePending},
								&memdb.StringFieldIndex{Field: keyTopic},
								&TimeFieldIndex{Field: keyScheduled},
							},
						},
					},
					indexActiveDeadline: {
						Name:         indexActiveDeadline,
						AllowMissing: true,
						Unique:       false,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&StateFieldIndex{Field: keyState, Filter: ratus.TaskStateActive},
								&TimeFieldIndex{Field: keyDeadline},
							},
						},
					},
					indexActiveTopic: {
						Name:         indexActiveTopic,
						AllowMissing: true,
						Unique:       false,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&StateFieldIndex{Field: keyState, Filter: ratus.TaskStateActive},
								&memdb.StringFieldIndex{Field: keyTopic},
							},
						},
					},
					indexCompletedConsumed: {
						Name:         indexCompletedConsumed,
						AllowMissing: true,
						Unique:       false,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&StateFieldIndex{Field: keyState, Filter: ratus.TaskStateCompleted},
								&TimeFieldIndex{Field: keyConsumed},
							},
						},
					},
				},
			},
		},
	}

	return &Engine{
		config: c,
		schema: &s,
	}, nil
}

// Open or connect to the storage engine.
func (g *Engine) Open(ctx context.Context) error {
	var err error
	g.database, err = memdb.NewMemDB(g.schema)
	return err
}

// Close or disconnect from the storage engine.
func (g *Engine) Close(ctx context.Context) error {
	// No need to close anything.
	return nil
}

// Destroy clears all data and closes the storage engine.
func (g *Engine) Destroy(ctx context.Context) error {
	if _, err := g.DeleteTopics(ctx); err != nil {
		return err
	}
	return g.Close(ctx)
}

// Ready probes the storage engine and returns an error if it is not ready.
func (g *Engine) Ready(ctx context.Context) error {
	if g.database == nil {
		return ratus.ErrServiceUnavailable
	}
	return nil
}
