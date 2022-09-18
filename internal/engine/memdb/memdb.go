// Package memdb implements the storage engine interface for MemDB.
package memdb

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/go-memdb"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/nonce"
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
	SnapshotPath     string        `arg:"--memdb-snapshot-path,env:MEMDB_SNAPSHOT_PATH" placeholder:"PATH" help:"path to the snapshot file" default:""`
	SnapshotInterval time.Duration `arg:"--memdb-snapshot-interval,env:MEMDB_SNAPSHOT_INTERVAL" placeholder:"DURATION" help:"interval for writing snapshots to disk" default:"5m"`

	RetentionPeriod time.Duration `arg:"--memdb-retention-period,env:MEMDB_RETENTION_PERIOD" placeholder:"DURATION" help:"retention period for completed tasks" default:"72h"`
}

// Engine implements the storage engine interface for MemDB.
type Engine struct {
	config   *Config
	schema   *memdb.DBSchema
	database *memdb.MemDB

	mux   sync.Mutex
	saved time.Time
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
	db, err := memdb.NewMemDB(g.schema)
	if err != nil {
		return err
	}

	// Load data from snapshot file if required.
	if g.config.SnapshotPath != "" {
		if err := load(db, g.config.SnapshotPath); err != nil {
			return err
		}
	}

	g.database = db
	return nil
}

// Close or disconnect from the storage engine.
func (g *Engine) Close(ctx context.Context) error {
	if g.config.SnapshotPath != "" {
		return save(g.database, g.config.SnapshotPath)
	}
	return nil
}

// Destroy clears all data and closes the storage engine.
func (g *Engine) Destroy(ctx context.Context) error {
	if _, err := g.DeleteTopics(ctx); err != nil {
		return err
	}
	if err := g.Close(ctx); err != nil {
		return err
	}

	// Remove the snapshot file after closing.
	if g.config.SnapshotPath != "" {
		return os.Remove(g.config.SnapshotPath)
	}

	return nil
}

// Ready probes the storage engine and returns an error if it is not ready.
func (g *Engine) Ready(ctx context.Context) error {
	if g.database == nil {
		return ratus.ErrServiceUnavailable
	}
	return nil
}

// updateOpsRecover returns a copy of the task with the state set back to
// "pending" and the nonce field cleared to invalidate subsequent commits.
func updateOpsRecover(v *ratus.Task) *ratus.Task {
	u := clone(v)
	u.State = ratus.TaskStatePending
	u.Nonce = ""
	return u
}

// updateOpsConsume returns a copy of the task with the state set to "active"
// and other fields populated with data from the promise.
func updateOpsConsume(v *ratus.Task, p *ratus.Promise, t time.Time) *ratus.Task {
	u := clone(v)
	u.State = ratus.TaskStateActive
	u.Nonce = nonce.Generate(ratus.NonceLength)
	u.Consumer = p.Consumer
	u.Consumed = &t
	u.Deadline = p.Deadline
	return u
}

// updateOpsCommit returns a copy of the task with updates specified in the
// commit applied to it.
func updateOpsCommit(v *ratus.Task, m *ratus.Commit) *ratus.Task {
	u := clone(v)
	u.Nonce = ""
	if m.Topic != "" {
		u.Topic = m.Topic
	}
	if m.State != nil {
		u.State = *m.State
	}
	if m.Scheduled != nil {
		u.Scheduled = m.Scheduled
	}
	if m.Payload != nil {
		u.Payload = m.Payload
	}
	return u
}

// clone returns a shallow copy of the data referenced by the specified pointer
// to avoid unsafe modifications of values in the database.
func clone[T any](v *T) *T {
	u := *v
	return &u
}

// save writes a snapshot of the database to a file.
func save(db *memdb.MemDB, path string) error {

	// Create a temporary file for writing the snapshot.
	p := fmt.Sprintf("%s.%s", path, nonce.Generate(8))
	f, err := os.Create(p)
	if err != nil {
		return err
	}

	// Rename or delete the temporary file before returning.
	var ok bool
	defer func() {
		f.Close()
		if !ok {
			os.Remove(p)
		} else {
			os.Rename(p, path)
		}
	}()

	// Create a snapshot of the database and encode all records.
	enc := gob.NewEncoder(f)
	txn := db.Snapshot().Txn(false)
	defer txn.Abort()
	it, err := txn.Get(tableTask, indexID)
	if err != nil {
		return err
	}
	for r := it.Next(); r != nil; r = it.Next() {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	txn.Commit()

	// Mark the operation as successful.
	ok = true

	return nil
}

// load recovers data from a snapshot file.
func load(db *memdb.MemDB, path string) error {

	// Open the snapshot file if it exists.
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	defer f.Close()

	// Decode records in the snapshot file and insert them into the database.
	dec := gob.NewDecoder(f)
	txn := db.Txn(true)
	defer txn.Abort()
	for {
		var t ratus.Task
		if err := dec.Decode(&t); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		if err := txn.Insert(tableTask, &t); err != nil {
			return err
		}
	}
	txn.Commit()

	return nil
}
