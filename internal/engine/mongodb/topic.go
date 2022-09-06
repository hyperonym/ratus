package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/hyperonym/ratus"
)

// ListTopics lists all topics.
func (g *Engine) ListTopics(ctx context.Context, limit, offset int) ([]*ratus.Topic, error) {

	// Use aggregation rather than the distinct command to support pagination.
	// This pipeline can use a DISTINCT_SCAN index plan that returns one
	// document per index key value.
	// https://www.mongodb.com/docs/v4.4/core/aggregation-pipeline-optimization/#indexes
	// https://www.mongodb.com/docs/v4.4/reference/operator/aggregation/group/#optimization-to-return-the-first-document-of-each-group
	p := mongo.Pipeline{
		bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$topic"}}}},
		bson.D{{Key: "$skip", Value: offset}},
		bson.D{{Key: "$limit", Value: limit}},
	}
	o := options.Aggregate().SetHint(hintTopic)
	r, err := g.collection.Aggregate(ctx, p, o)
	if err != nil {
		return nil, err
	}

	// For performance reasons, the aggregated results do not include the
	// number of tasks under each topic.
	v := make([]*ratus.Topic, 0)
	if err := r.All(ctx, &v); err != nil {
		return nil, err
	}

	return v, nil
}

// DeleteTopics deletes all topics and tasks.
func (g *Engine) DeleteTopics(ctx context.Context) (*ratus.Deleted, error) {
	f := bson.D{}
	o := options.Delete().SetHint(hintID)
	r, err := g.collection.DeleteMany(ctx, f, o)
	if err != nil {
		return nil, err
	}

	// Return the number of deleted tasks, not the number of deleted topics.
	return &ratus.Deleted{
		Deleted: r.DeletedCount,
	}, nil
}

// GetTopic gets information about a topic.
func (g *Engine) GetTopic(ctx context.Context, topic string) (*ratus.Topic, error) {

	// Get the number of tasks under the topic.
	f := bson.D{{Key: "topic", Value: topic}}
	o := options.Count().SetHint(hintTopic)
	n, err := g.collection.CountDocuments(ctx, f, o)

	// Topics are not created manually, their existence depends entirely on
	// whether there are tasks with the corresponding topic properties.
	if err == nil && n == 0 {
		err = ratus.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Currently the information of a topic only contains the name and count.
	// Information such as progress and states could be added in the future.
	return &ratus.Topic{
		Name:  topic,
		Count: n,
	}, nil
}

// DeleteTopic deletes a topic and its tasks.
func (g *Engine) DeleteTopic(ctx context.Context, topic string) (*ratus.Deleted, error) {
	f := bson.D{{Key: "topic", Value: topic}}
	o := options.Delete().SetHint(hintTopic)
	r, err := g.collection.DeleteMany(ctx, f, o)
	if err != nil {
		return nil, err
	}

	// Return the number of deleted tasks, not the number of deleted topics.
	return &ratus.Deleted{
		Deleted: r.DeletedCount,
	}, nil
}
