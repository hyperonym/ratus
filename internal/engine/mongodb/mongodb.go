// Package mongodb implements the storage engine interface for MongoDB.
package mongodb

import (
	"time"

	"go.mongodb.org/mongo-driver/mongo"
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
}
