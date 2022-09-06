package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/hyperonym/ratus"
)

// Chore recovers timed out tasks and deletes expired tasks.
func (g *Engine) Chore(ctx context.Context) error {

	// Find all active tasks whose deadline is before the current time.
	f := bson.D{
		{Key: keyState, Value: ratus.TaskStateActive},
		{Key: keyDeadline, Value: bson.D{
			{Key: "$lt", Value: time.Now()},
		}},
	}

	// Recover tasks that have timed out.
	o := options.Update().SetUpsert(false).SetHint(hintActiveDeadline)
	if _, err := g.collection.UpdateMany(ctx, f, updateOpsRecover(), o); err != nil {
		return err
	}

	// Deletion of expired tasks is handled by the TTL index automatically.
	return nil
}

// Poll claims the next available task in the topic based on the scheduled time.
func (g *Engine) Poll(ctx context.Context, topic string, p *ratus.Promise) (*ratus.Task, error) {
	return branch(func() (*ratus.Task, error) {
		return g.pollAtomic(ctx, topic, p)
	}, func() (*ratus.Task, error) {
		return g.pollOptimistic(ctx, topic, p)
	}, g.fallbackPoll)
}

// pollAtomic is the preferred implementation of Poll.
func (g *Engine) pollAtomic(ctx context.Context, topic string, p *ratus.Promise) (*ratus.Task, error) {

	// Use an atomic findAndModify command to secure the next task in topic if
	// available, and return the updated task. This operation is expected to
	// work only on unsharded collections and sharded collections using the
	// topic field as the shard key.
	var v ratus.Task
	t := time.Now()
	f := queryOpsPoll(topic, t)
	u := updateOpsConsume(p, t)
	s := bson.D{{Key: keyScheduled, Value: 1}}
	o := options.FindOneAndUpdate().SetUpsert(false).SetSort(s).SetReturnDocument(options.After).SetHint(hintPendingTopicScheduled)
	if err := g.collection.FindOneAndUpdate(ctx, f, u, o).Decode(&v); err != nil {
		if err == mongo.ErrNoDocuments {
			err = ratus.ErrNotFound
		}
		return nil, err
	}

	return &v, nil
}

// pollOptimistic is the fallback implementation of Poll.
func (g *Engine) pollOptimistic(ctx context.Context, topic string, p *ratus.Promise) (*ratus.Task, error) {

	// Peek into the topic to get the ID and nonce of the next candidate task.
	t := time.Now()
	f := queryOpsPoll(topic, t)
	s := bson.D{{Key: keyScheduled, Value: 1}}
	c, err := g.peek(ctx, f, s, hintPendingTopicScheduled)
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
	f = append(f, bson.E{Key: keyID, Value: c.ID})
	f = append(f, bson.E{Key: keyNonce, Value: c.Nonce})
	u := updateOpsConsume(p, t)
	n := options.FindOneAndUpdate().SetUpsert(false).SetReturnDocument(options.After).SetHint(hintID)
	if err := g.collection.FindOneAndUpdate(ctx, f, u, n).Decode(&v); err != nil {

		// The only reason that could lead to no match is that the task has
		// been obtained by another consumer. Retry immediately to secure the
		// next task!
		if err == mongo.ErrNoDocuments {
			return g.pollOptimistic(ctx, topic, p)
		}
		return nil, err
	}

	return &v, nil
}

// Commit applies a set of updates to a task and returns the updated task.
func (g *Engine) Commit(ctx context.Context, id string, m *ratus.Commit) (*ratus.Task, error) {
	return branch(func() (*ratus.Task, error) {
		return g.commitAtomic(ctx, id, m)
	}, func() (*ratus.Task, error) {
		return g.commitOptimistic(ctx, id, m)
	}, g.fallbackCommit)
}

// commitAtomic is the preferred implementation of Commit.
func (g *Engine) commitAtomic(ctx context.Context, id string, m *ratus.Commit) (*ratus.Task, error) {

	// Verify the nonce if provided to invalidate unintended commits.
	var v ratus.Task
	f := bson.D{{Key: keyID, Value: id}}
	if m.Nonce != "" {
		f = append(f, bson.E{Key: keyNonce, Value: m.Nonce})
	}
	u := updateOpsCommit(m)
	o := options.FindOneAndUpdate().SetUpsert(false).SetReturnDocument(options.After).SetHint(hintID)

	// Use an atomic findAndModify command to apply the updates and return the
	// update task. This operation is expected to work only on unsharded
	// collections and sharded collections using the ID field as the shard key.
	if err := g.collection.FindOneAndUpdate(ctx, f, u, o).Decode(&v); err != nil {

		// Check if the failure is due to a mismatch of nonce or the target
		// task does not exist.
		if err == mongo.ErrNoDocuments {
			if m.Nonce != "" && g.exists(ctx, bson.D{{Key: keyID, Value: id}}, hintID) {
				err = ratus.ErrConflict
			} else {
				err = ratus.ErrNotFound
			}
		}
		return nil, err
	}

	return &v, nil
}

// commitOptimistic is the fallback implementation of Commit.
func (g *Engine) commitOptimistic(ctx context.Context, id string, m *ratus.Commit) (*ratus.Task, error) {

	// Get current information of the target task.
	f := bson.D{{Key: keyID, Value: id}}
	c, err := g.peek(ctx, f, nil, hintID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			err = ratus.ErrNotFound
		}
		return nil, err
	}

	// Verify the nonce if provided to invalidate unintended commits.
	if m.Nonce != "" && m.Nonce != c.Nonce {
		return nil, ratus.ErrConflict
	}

	// Add all known fields to the filter criteria to perform findAndModify.
	// This operation is expected to work on sharded collections using various
	// sharding strategies.
	var v ratus.Task
	f = append(f, bson.E{Key: keyTopic, Value: c.Topic})
	f = append(f, bson.E{Key: keyState, Value: c.State})
	f = append(f, bson.E{Key: keyNonce, Value: c.Nonce})
	u := updateOpsCommit(m)
	n := options.FindOneAndUpdate().SetUpsert(false).SetReturnDocument(options.After).SetHint(hintID)
	if err := g.collection.FindOneAndUpdate(ctx, f, u, n).Decode(&v); err != nil {

		// The only reason that could lead to no match is that the task has
		// been updated by another consumer. Therefore it is considered to be
		// a conflict.
		if err == mongo.ErrNoDocuments {
			err = ratus.ErrConflict
		}
		return nil, err
	}

	return &v, nil
}
