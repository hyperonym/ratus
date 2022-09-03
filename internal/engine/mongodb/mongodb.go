// Package mongodb implements the storage engine interface for MongoDB.
package mongodb

import (
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Config contains configurations for the MongoDB storage engine.
type Config struct {
	URI        string `arg:"--mongodb-uri,env:MONGODB_URI" placeholder:"URI" help:"connection URI of the MongoDB deployment to connect to" default:"mongodb://127.0.0.1:27017"`
	Database   string `arg:"--mongodb-database,env:MONGODB_DATABASE" placeholder:"NAME" help:"name of the MongoDB database to use" default:"ratus"`
	Collection string `arg:"--mongodb-collection,env:MONGODB_COLLECTION" placeholder:"NAME" help:"name of the MongoDB collection to store tasks" default:"tasks"`

	RetentionPeriod time.Duration `arg:"--mongodb-retention-period,env:MONGODB_RETENTION_PERIOD" placeholder:"DURATION" help:"retention period for completed tasks" default:"72h"`

	DisableIndexCreation bool `arg:"--mongodb-disable-index-creation,env:MONGODB_DISABLE_INDEX_CREATION" help:"disable automatic index creation on startup"`
	DisableAutoFallback  bool `arg:"--mongodb-disable-auto-fallback,env:MONGODB_DISABLE_AUTO_FALLBACK" help:"disable transparent fallbacks for unsupported operations"`
	DisableAtomicPoll    bool `arg:"--mongodb-disable-atomic-poll,env:MONGODB_DISABLE_ATOMIC_POLL" help:"disable atomic polling and fallback to optimistic concurrency control"`
}

// Engine implements the storage engine interface for MongoDB.
type Engine struct {
	config     *Config
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection

	// Atomic fallback flags: -1 = disabled, 0 = auto, 1 = enabled.
	fallbackPoll          *atomic.Int32
	fallbackCommit        *atomic.Int32
	fallbackUpsertTasks   *atomic.Int32
	fallbackUpsertTask    *atomic.Int32
	fallbackInsertPromise *atomic.Int32
	fallbackUpsertPromise *atomic.Int32
}

// New creates a new MongoDB storage engine instance.
func New(c *Config) (*Engine, error) {
	g := Engine{
		config:                c,
		fallbackPoll:          &atomic.Int32{},
		fallbackCommit:        &atomic.Int32{},
		fallbackUpsertTasks:   &atomic.Int32{},
		fallbackUpsertTask:    &atomic.Int32{},
		fallbackInsertPromise: &atomic.Int32{},
		fallbackUpsertPromise: &atomic.Int32{},
	}

	// Create a new client without actually connecting to the deployment.
	// Initialization processes that requires I/O should happen in Open.
	var err error
	g.client, err = mongo.NewClient(options.Client().ApplyURI(c.URI))
	if err != nil {
		return nil, err
	}

	// Get handles for the database and the collection.
	g.database = g.client.Database(c.Database)
	g.collection = g.database.Collection(c.Collection)

	// Disable transparent fallbacks if required.
	if c.DisableAutoFallback {
		g.Fallback(-1)
	}

	// Disable atomic polling if required.
	if c.DisableAtomicPoll {
		g.fallbackPoll.Store(1)
	}

	return &g, nil
}

// Collection returns the handle for the task collection.
func (g *Engine) Collection() *mongo.Collection {
	return g.collection
}

// Fallback sets all fallback flags to the given value.
func (g *Engine) Fallback(v int32) *Engine {
	g.fallbackPoll.Store(v)
	g.fallbackCommit.Store(v)
	g.fallbackUpsertTasks.Store(v)
	g.fallbackUpsertTask.Store(v)
	g.fallbackInsertPromise.Store(v)
	g.fallbackUpsertPromise.Store(v)
	return g
}
