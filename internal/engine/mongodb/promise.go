package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/hyperonym/ratus"
)

// ListPromises lists all promises in a topic.
func (g *Engine) ListPromises(ctx context.Context, topic string, limit, offset int) ([]*ratus.Promise, error) {
	f := bson.D{
		{Key: keyState, Value: ratus.TaskStateActive},
		{Key: keyTopic, Value: topic},
	}

	// Promises in effect are represented in MongoDB as fields of the active tasks.
	o := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset)).SetHint(hintActiveTopic)
	r, err := g.collection.Find(ctx, f, o)
	if err != nil {
		return nil, err
	}
	v := make([]*ratus.Promise, 0)
	if err := r.All(ctx, &v); err != nil {
		return nil, err
	}

	return v, nil
}

// DeletePromises deletes all promises in a topic.
func (g *Engine) DeletePromises(ctx context.Context, topic string) (*ratus.Deleted, error) {
	f := bson.D{
		{Key: keyState, Value: ratus.TaskStateActive},
		{Key: keyTopic, Value: topic},
	}

	// Deleting promises is equivalent to setting the states of the active
	// tasks back to "pending" and clearing the nonce fields.
	o := options.Update().SetUpsert(false).SetHint(hintActiveTopic)
	r, err := g.collection.UpdateMany(ctx, f, updateOpsRecover(), o)
	if err != nil {
		return nil, err
	}

	return &ratus.Deleted{
		Deleted: r.ModifiedCount,
	}, nil
}

// GetPromise gets a promise by the unique ID of its target task.
func (g *Engine) GetPromise(ctx context.Context, id string) (*ratus.Promise, error) {
	var v ratus.Promise
	f := bson.D{
		{Key: keyID, Value: id},
		{Key: keyState, Value: ratus.TaskStateActive},
	}

	// A promise in effect is represented in MongoDB as fields of an active task.
	o := options.FindOne().SetHint(hintID)
	if err := g.collection.FindOne(ctx, f, o).Decode(&v); err != nil {
		if err == mongo.ErrNoDocuments {
			err = ratus.ErrNotFound
		}
		return nil, err
	}

	return &v, nil
}

// InsertPromise claims the target task if it is in pending state.
func (g *Engine) InsertPromise(ctx context.Context, p *ratus.Promise) (*ratus.Task, error) {
	return branch(func() (*ratus.Task, error) {
		return g.insertPromiseAtomic(ctx, p)
	}, func() (*ratus.Task, error) {
		return g.insertPromiseOptimistic(ctx, p)
	}, g.fallbackInsertPromise)
}

// insertPromiseAtomic is the preferred implementation of InsertPromise.
func (g *Engine) insertPromiseAtomic(ctx context.Context, p *ratus.Promise) (*ratus.Task, error) {

	// Use an atomic findAndModify command to claim and return the task.
	// This operation is expected to work only on unsharded collections and
	// sharded collections using the ID field as the shard key.
	var v ratus.Task
	t := time.Now()
	f := bson.D{
		{Key: keyID, Value: p.ID},
		{Key: keyState, Value: ratus.TaskStatePending},
	}
	u := updateOpsConsume(p, t)
	o := options.FindOneAndUpdate().SetUpsert(false).SetReturnDocument(options.After).SetHint(hintID)
	if err := g.collection.FindOneAndUpdate(ctx, f, u, o).Decode(&v); err != nil {

		// Check if the failure is due to a mismatch of state or the target
		// task does not exist.
		if err == mongo.ErrNoDocuments {
			if g.exists(ctx, bson.D{{Key: keyID, Value: p.ID}}, hintID) {
				err = ratus.ErrConflict
			} else {
				err = ratus.ErrNotFound
			}
		}
		return nil, err
	}

	return &v, nil
}

// insertPromiseOptimistic is the fallback implementation of InsertPromise.
func (g *Engine) insertPromiseOptimistic(ctx context.Context, p *ratus.Promise) (*ratus.Task, error) {

	// Get current information of the target task.
	f := bson.D{{Key: keyID, Value: p.ID}}
	c, err := g.peek(ctx, f, nil, hintID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			err = ratus.ErrNotFound
		}
		return nil, err
	}

	// Check if the target task is in pending state.
	if c.State != ratus.TaskStatePending {
		return nil, ratus.ErrConflict
	}

	// Add all known fields to the filter criteria to perform findAndModify.
	// This operation is expected to work on sharded collections using various
	// sharding strategies.
	var v ratus.Task
	t := time.Now()
	f = append(f, bson.E{Key: keyTopic, Value: c.Topic})
	f = append(f, bson.E{Key: keyState, Value: ratus.TaskStatePending})
	f = append(f, bson.E{Key: keyNonce, Value: c.Nonce})
	u := updateOpsConsume(p, t)
	n := options.FindOneAndUpdate().SetUpsert(false).SetReturnDocument(options.After).SetHint(hintID)
	if err := g.collection.FindOneAndUpdate(ctx, f, u, n).Decode(&v); err != nil {

		// The only reason that could lead to no match is that another consumer
		// has obtained the task.
		if err == mongo.ErrNoDocuments {
			err = ratus.ErrConflict
		}
		return nil, err
	}

	return &v, nil
}

// UpsertPromise claims the target task regardless of its current state.
func (g *Engine) UpsertPromise(ctx context.Context, p *ratus.Promise) (*ratus.Task, error) {
	return branch(func() (*ratus.Task, error) {
		return g.upsertPromiseAtomic(ctx, p)
	}, func() (*ratus.Task, error) {
		return g.upsertPromiseOptimistic(ctx, p)
	}, g.fallbackUpsertPromise)
}

// upsertPromiseAtomic is the preferred implementation of UpsertPromise.
func (g *Engine) upsertPromiseAtomic(ctx context.Context, p *ratus.Promise) (*ratus.Task, error) {

	// Use an atomic findAndModify command to claim and return the task.
	// This operation is expected to work only on unsharded collections and
	// sharded collections using the ID field as the shard key.
	var v ratus.Task
	t := time.Now()
	f := bson.D{{Key: keyID, Value: p.ID}}
	u := updateOpsConsume(p, t)
	o := options.FindOneAndUpdate().SetUpsert(false).SetReturnDocument(options.After).SetHint(hintID)
	if err := g.collection.FindOneAndUpdate(ctx, f, u, o).Decode(&v); err != nil {
		if err == mongo.ErrNoDocuments {
			err = ratus.ErrNotFound
		}
		return nil, err
	}

	return &v, nil
}

// upsertPromiseOptimistic is the fallback implementation of UpsertPromise.
func (g *Engine) upsertPromiseOptimistic(ctx context.Context, p *ratus.Promise) (*ratus.Task, error) {

	// Get current information of the target task.
	f := bson.D{{Key: keyID, Value: p.ID}}
	c, err := g.peek(ctx, f, nil, hintID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			err = ratus.ErrNotFound
		}
		return nil, err
	}

	// Add all known fields to the filter criteria to perform findAndModify.
	// This operation is expected to work on sharded collections using various
	// sharding strategies.
	var v ratus.Task
	t := time.Now()
	f = append(f, bson.E{Key: keyTopic, Value: c.Topic})
	f = append(f, bson.E{Key: keyState, Value: c.State})
	f = append(f, bson.E{Key: keyNonce, Value: c.Nonce})
	u := updateOpsConsume(p, t)
	n := options.FindOneAndUpdate().SetUpsert(false).SetReturnDocument(options.After).SetHint(hintID)
	if err := g.collection.FindOneAndUpdate(ctx, f, u, n).Decode(&v); err != nil {

		// The only reason that could lead to no match is that another consumer
		// has obtained the task. Then retry immediately.
		if err == mongo.ErrNoDocuments {
			return g.upsertPromiseOptimistic(ctx, p)
		}
		return nil, err
	}

	return &v, nil
}

// DeletePromise deletes a promise by the unique ID of its target task.
func (g *Engine) DeletePromise(ctx context.Context, id string) (*ratus.Deleted, error) {
	f := bson.D{
		{Key: keyID, Value: id},
		{Key: keyState, Value: ratus.TaskStateActive},
	}

	// Deleting a promise is equivalent to setting the state of the target task
	// back to "pending" and clearing the nonce field.
	o := options.Update().SetUpsert(false).SetHint(hintID)
	r, err := g.collection.UpdateOne(ctx, f, updateOpsRecover(), o)
	if err != nil {
		return nil, err
	}

	return &ratus.Deleted{
		Deleted: r.ModifiedCount,
	}, nil
}
