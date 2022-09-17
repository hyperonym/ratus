package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/hyperonym/ratus"
)

// ListTasks lists all tasks in a topic.
func (g *Engine) ListTasks(ctx context.Context, topic string, limit, offset int) ([]*ratus.Task, error) {
	f := bson.D{{Key: keyTopic, Value: topic}}
	o := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset)).SetHint(indexTopic)
	r, err := g.collection.Find(ctx, f, o)
	if err != nil {
		return nil, err
	}
	v := make([]*ratus.Task, 0)
	if err := r.All(ctx, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// InsertTasks inserts a batch of tasks while ignoring existing ones.
func (g *Engine) InsertTasks(ctx context.Context, ts []*ratus.Task) (*ratus.Updated, error) {
	w := make([]mongo.WriteModel, len(ts))
	for i, t := range ts {
		m := mongo.NewInsertOneModel()
		m = m.SetDocument(t)
		w[i] = m
	}

	// Execute an unordered bulk write to insert tasks and ignore duplicates.
	// If an error occurs during the processing of one of the write operations,
	// MongoDB will continue to process remaining write operations in the list.
	o := options.BulkWrite().SetOrdered(false)
	r, err := g.collection.BulkWrite(ctx, w, o)
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return nil, err
	}

	return &ratus.Updated{
		Created: r.InsertedCount + r.UpsertedCount,
		Updated: r.ModifiedCount,
	}, nil
}

// UpsertTasks inserts or updates a batch of tasks.
func (g *Engine) UpsertTasks(ctx context.Context, ts []*ratus.Task) (*ratus.Updated, error) {
	return branch(func() (*ratus.Updated, error) {
		return g.upsertTasksReplace(ctx, ts)
	}, func() (*ratus.Updated, error) {
		return g.upsertTasksDeleteAndInsert(ctx, ts)
	}, g.fallbackUpsertTasks)
}

// upsertTasksReplace is the preferred implementation of UpsertTasks.
func (g *Engine) upsertTasksReplace(ctx context.Context, ts []*ratus.Task) (*ratus.Updated, error) {

	// Perform replace operation with upsert enabled for each task in the list.
	// This operation is expected to work only on unsharded collections and
	// sharded collections using the ID field as the shard key.
	w := make([]mongo.WriteModel, len(ts))
	for i, t := range ts {
		m := mongo.NewReplaceOneModel()
		m = m.SetFilter(bson.D{{Key: keyID, Value: t.ID}})
		m = m.SetReplacement(t)
		m = m.SetUpsert(true)
		m = m.SetHint(indexID)
		w[i] = m
	}
	o := options.BulkWrite().SetOrdered(false)
	r, err := g.collection.BulkWrite(ctx, w, o)
	if err != nil {
		return nil, err
	}

	return &ratus.Updated{
		Created: r.InsertedCount + r.UpsertedCount,
		Updated: r.ModifiedCount,
	}, nil
}

// upsertTasksDeleteAndInsert is the fallback implementation of UpsertTasks.
func (g *Engine) upsertTasksDeleteAndInsert(ctx context.Context, ts []*ratus.Task) (*ratus.Updated, error) {

	// Delete tasks with the same IDs before inserting to avoid modification of
	// shard key values. It's ugly, but as far as I know it's the only way to
	// circumvent MongoDB's own limitations on sharded collections:
	// https://www.mongodb.com/docs/v4.4/reference/method/db.collection.replaceOne/#shard-key-modification
	w := make([]mongo.WriteModel, len(ts))
	for i, t := range ts {
		m := mongo.NewDeleteOneModel()
		m = m.SetFilter(bson.D{{Key: keyID, Value: t.ID}})
		m = m.SetHint(indexID)
		w[i] = m
	}

	// The number of deleted tasks is the number of tasks that should be updated.
	o := options.BulkWrite().SetOrdered(false)
	r, err := g.collection.BulkWrite(ctx, w, o)
	if err != nil {
		return nil, err
	}

	// Insert the tasks and ignore duplicate key errors in race conditions.
	// This operation is expected to work on sharded collections using various
	// sharding strategies.
	if _, err := g.InsertTasks(ctx, ts); err != nil {
		return nil, err
	}

	return &ratus.Updated{
		Created: int64(len(ts)) - r.DeletedCount,
		Updated: r.DeletedCount,
	}, nil
}

// DeleteTasks deletes all tasks in a topic.
func (g *Engine) DeleteTasks(ctx context.Context, topic string) (*ratus.Deleted, error) {
	f := bson.D{{Key: keyTopic, Value: topic}}
	o := options.Delete().SetHint(indexTopic)
	r, err := g.collection.DeleteMany(ctx, f, o)
	if err != nil {
		return nil, err
	}
	return &ratus.Deleted{
		Deleted: r.DeletedCount,
	}, nil
}

// GetTask gets a task by its unique ID.
func (g *Engine) GetTask(ctx context.Context, id string) (*ratus.Task, error) {
	var v ratus.Task
	f := bson.D{{Key: keyID, Value: id}}
	o := options.FindOne().SetAllowPartialResults(true).SetHint(indexID)
	if err := g.collection.FindOne(ctx, f, o).Decode(&v); err != nil {
		if err == mongo.ErrNoDocuments {
			err = ratus.ErrNotFound
		}
		return nil, err
	}
	return &v, nil
}

// InsertTask inserts a new task.
func (g *Engine) InsertTask(ctx context.Context, t *ratus.Task) (*ratus.Updated, error) {
	if _, err := g.collection.InsertOne(ctx, t); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			err = ratus.ErrConflict
		}
		return nil, err
	}
	return &ratus.Updated{
		Created: 1,
		Updated: 0,
	}, nil
}

// UpsertTask inserts or updates a task.
func (g *Engine) UpsertTask(ctx context.Context, t *ratus.Task) (*ratus.Updated, error) {
	return branch(func() (*ratus.Updated, error) {
		return g.upsertTaskReplace(ctx, t)
	}, func() (*ratus.Updated, error) {
		return g.upsertTasksDeleteAndInsert(ctx, []*ratus.Task{t})
	}, g.fallbackUpsertTask)
}

// upsertTaskReplace is the preferred implementation of UpsertTask.
func (g *Engine) upsertTaskReplace(ctx context.Context, t *ratus.Task) (*ratus.Updated, error) {

	// Perform replace operation with upsert enabled.
	// This operation is expected to work only on unsharded collections and
	// sharded collections using the ID field as the shard key.
	f := bson.D{{Key: keyID, Value: t.ID}}
	o := options.Replace().SetUpsert(true).SetHint(indexID)
	r, err := g.collection.ReplaceOne(ctx, f, t, o)
	if err != nil {
		return nil, err
	}

	return &ratus.Updated{
		Created: r.UpsertedCount,
		Updated: r.ModifiedCount,
	}, nil
}

// DeleteTask deletes a task by its unique ID.
func (g *Engine) DeleteTask(ctx context.Context, id string) (*ratus.Deleted, error) {
	f := bson.D{{Key: keyID, Value: id}}
	o := options.Delete().SetHint(indexID)
	r, err := g.collection.DeleteOne(ctx, f, o)
	if err != nil {
		return nil, err
	}
	return &ratus.Deleted{
		Deleted: r.DeletedCount,
	}, nil
}
