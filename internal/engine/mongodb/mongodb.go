// Package mongodb implements the storage engine interface for MongoDB.
package mongodb

import (
	"context"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golang.org/x/sync/errgroup"

	"github.com/hyperonym/ratus"
)

// Name constants for index creation and selection.
const (
	hintID                    = "_id_"
	hintTopic                 = "topic_hashed"
	hintPendingTopicScheduled = "topic_1_scheduled_1"
	hintActiveDeadline        = "deadline_1"
	hintActiveTopic           = "topic_1"
	hintCompletedConsumed     = "consumed_1"
)

// Partial filter expressions for index creation.
var (
	filterStatePending   = bson.D{{Key: "state", Value: ratus.TaskStatePending}}
	filterStateActive    = bson.D{{Key: "state", Value: ratus.TaskStateActive}}
	filterStateCompleted = bson.D{{Key: "state", Value: ratus.TaskStateCompleted}}
)

// Config contains configurations for the MongoDB storage engine.
type Config struct {
	URI        string `arg:"--mongodb-uri,env:MONGODB_URI" placeholder:"URI" help:"connection URI of the MongoDB deployment to connect to" default:"mongodb://127.0.0.1:27017"`
	Database   string `arg:"--mongodb-database,env:MONGODB_DATABASE" placeholder:"NAME" help:"name of the MongoDB database to use" default:"ratus"`
	Collection string `arg:"--mongodb-collection,env:MONGODB_COLLECTION" placeholder:"NAME" help:"name of the MongoDB collection to store tasks" default:"tasks"`

	RetentionPeriod time.Duration `arg:"--mongodb-retention-period,env:MONGODB_RETENTION_PERIOD" placeholder:"DURATION" help:"retention period for completed tasks" default:"72h"`

	DisableIndexCreation bool `arg:"--mongodb-disable-index-creation,env:MONGODB_DISABLE_INDEX_CREATION" help:"disable automatic index creation on startup"`
	DisableAutoFallback  bool `arg:"--mongodb-disable-auto-fallback,env:MONGODB_DISABLE_AUTO_FALLBACK" help:"disable transparent fallbacks for unsupported operations"`
	DisableAtomicPoll    bool `arg:"--mongodb-disable-atomic-poll,env:MONGODB_DISABLE_ATOMIC_POLL" help:"disable atomic polling and fallback to optimistic locking"`
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

// Open or connect to the storage engine.
func (g *Engine) Open(ctx context.Context) error {

	// Connect to the deployment but do not use Ping to verify the connection
	// as it reduces application resilience because applications starting up
	// will error if the server is temporarily unavailable or is failing over.
	if err := g.client.Connect(ctx); err != nil {
		return err
	}

	// Create indexes on the collection if required.
	if !g.config.DisableIndexCreation {
		if err := g.createIndexes(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Close or disconnect from the storage engine.
func (g *Engine) Close(ctx context.Context) error {
	return g.client.Disconnect(ctx)
}

// Destroy clears all data and closes the storage engine.
func (g *Engine) Destroy(ctx context.Context) error {
	if err := g.collection.Drop(ctx); err != nil {
		return err
	}
	return g.Close(ctx)
}

// Ready probes the storage engine and returns an error if it is not ready.
func (g *Engine) Ready(ctx context.Context) error {
	if err := g.client.Ping(ctx, readpref.Primary()); err != nil {
		return ratus.ErrServiceUnavailable
	}
	return nil
}

// createIndexes creates all indexes required for queue operations.
func (g *Engine) createIndexes(ctx context.Context) error {
	v := g.collection.Indexes()
	e, ctx := errgroup.WithContext(ctx)

	// Create indexes that do not require TTL settings.
	e.Go(func() error {
		_, err := v.CreateMany(ctx, []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "topic", Value: "hashed"}},
				Options: options.Index().SetName(hintTopic),
			},
			{
				Keys:    bson.D{{Key: "topic", Value: 1}, {Key: "scheduled", Value: 1}},
				Options: options.Index().SetName(hintPendingTopicScheduled).SetPartialFilterExpression(filterStatePending),
			},
			{
				Keys:    bson.D{{Key: "deadline", Value: 1}},
				Options: options.Index().SetName(hintActiveDeadline).SetPartialFilterExpression(filterStateActive),
			},
			{
				Keys:    bson.D{{Key: "topic", Value: 1}},
				Options: options.Index().SetName(hintActiveTopic).SetPartialFilterExpression(filterStateActive),
			},
		})
		return err
	})

	// Create TTL index to automatically delete completed tasks that have
	// exceeded their retention period.
	e.Go(func() error {
		k := bson.D{{Key: "consumed", Value: 1}}
		s := int32(g.config.RetentionPeriod.Seconds())

		// Attempt to create a new TTL index. This operation will fail if the
		// specified TTL value does not match the value in the existing index.
		_, err := v.CreateOne(ctx, mongo.IndexModel{
			Keys:    k,
			Options: options.Index().SetName(hintCompletedConsumed).SetPartialFilterExpression(filterStateCompleted).SetExpireAfterSeconds(s),
		})
		if err == nil {
			return nil
		}
		if c, ok := err.(mongo.CommandError); !ok || c.Name != "IndexOptionsConflict" {
			return err
		}

		// Use the collMod command in conjunction with the index collection
		// flag to change the value of expireAfterSeconds of an existing index.
		if err := g.database.RunCommand(ctx, bson.D{
			{Key: "collMod", Value: g.collection.Name()},
			{Key: "index", Value: bson.D{
				{Key: "keyPattern", Value: k},
				{Key: "expireAfterSeconds", Value: s},
			}},
		}).Err(); err != nil && err != mongo.ErrNoDocuments {
			return err
		}

		return nil
	})

	return e.Wait()
}

// updateOpsRecover returns a document containing update operators to set the
// state of the tasks back to "pending" and clear the nonce field to invalidate
// subsequent commits.
func updateOpsRecover() bson.D {
	return bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "state", Value: ratus.TaskStatePending},
			{Key: "nonce", Value: ""},
		}},
	}
}
