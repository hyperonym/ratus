package memdb

import (
	"context"

	"github.com/hyperonym/ratus"
)

// ListTopics lists all topics.
func (g *Engine) ListTopics(ctx context.Context, limit, offset int) ([]*ratus.Topic, error) {
	txn := g.database.Txn(false)
	defer txn.Abort()

	// Listing topics is a very expensive operation as it requires scanning the
	// entire database until the required number of results are collected.
	var (
		p string
		n int
	)
	it, err := txn.Get(tableTask, indexTopic)
	if err != nil {
		return nil, err
	}
	v := make([]*ratus.Topic, 0)
	for r := it.Next(); r != nil && len(v) < limit; r = it.Next() {
		if t := r.(*ratus.Task); t.Topic != p {
			p = t.Topic
			if n >= offset {
				v = append(v, &ratus.Topic{Name: t.Topic})
			}
			n++
		}
	}

	txn.Commit()
	return v, nil
}

// DeleteTopics deletes all topics and tasks.
func (g *Engine) DeleteTopics(ctx context.Context) (*ratus.Deleted, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	// Return the number of deleted tasks, not the number of deleted topics.
	n, err := txn.DeleteAll(tableTask, indexTopic)
	if err != nil {
		return nil, err
	}

	txn.Commit()
	return &ratus.Deleted{
		Deleted: int64(n),
	}, nil
}

// GetTopic gets information about a topic.
func (g *Engine) GetTopic(ctx context.Context, topic string) (*ratus.Topic, error) {
	txn := g.database.Txn(false)
	defer txn.Abort()

	// Count records by introspecting the underlying radix tree. Reference:
	// https://github.com/hashicorp/go-memdb/issues/83#issuecomment-1168332874
	var n int64
	it, err := txn.Get(tableTask, indexTopic, topic)
	if err != nil {
		return nil, err
	}
	for r := it.Next(); r != nil; r = it.Next() {
		n++
	}
	if n == 0 {
		return nil, ratus.ErrNotFound
	}

	// Currently the information of a topic only contains the name and count.
	// Information such as progress and states could be added in the future.
	txn.Commit()
	return &ratus.Topic{
		Name:  topic,
		Count: n,
	}, nil
}

// DeleteTopic deletes a topic and its tasks.
func (g *Engine) DeleteTopic(ctx context.Context, topic string) (*ratus.Deleted, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	// Return the number of deleted tasks, not the number of deleted topics.
	n, err := txn.DeleteAll(tableTask, indexTopic, topic)
	if err != nil {
		return nil, err
	}

	txn.Commit()
	return &ratus.Deleted{
		Deleted: int64(n),
	}, nil
}
