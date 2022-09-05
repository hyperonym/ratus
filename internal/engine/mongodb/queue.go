package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
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
