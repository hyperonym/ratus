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
		{Key: "state", Value: ratus.TaskStateActive},
		{Key: "deadline", Value: bson.D{
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
	s := bson.D{{Key: "scheduled", Value: 1}}
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
	s := bson.D{{Key: "scheduled", Value: 1}}
	c, err := g.peek(ctx, f, s, hintPendingTopicScheduled)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			err = ratus.ErrNotFound
		}
		return nil, err
	}

	// Add the unique ID and nonce of the candidate task to the filter criteria
	// to implement optimistic concurrency control. This operation is expected
	// to work on sharded collections using various sharding strategies.
	var v ratus.Task
	f = append(f, bson.E{Key: "_id", Value: c.ID})
	f = append(f, bson.E{Key: "nonce", Value: c.Nonce})
	u := updateOpsConsume(p, t)
	n := options.FindOneAndUpdate().SetUpsert(false).SetReturnDocument(options.After).SetHint(hintID)
	if err := g.collection.FindOneAndUpdate(ctx, f, u, n).Decode(&v); err != nil {

		// The only reason that could lead to no match is that another consumer
		// has obtained the task. Retry immediately to secure the next task!
		if err == mongo.ErrNoDocuments {
			return g.pollOptimistic(ctx, topic, p)
		}
		return nil, err
	}

	return &v, nil
}
