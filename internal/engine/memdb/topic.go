package memdb

import (
	"context"

	"github.com/hyperonym/ratus"
)

// ListTopics lists all topics.
func (g *Engine) ListTopics(ctx context.Context, limit, offset int) ([]*ratus.Topic, error) {
	txn := g.database.Txn(false)
	defer txn.Abort()

	// Listing all topics is a very expensive operation as it requires scanning
	// the entire database until the required number of results are collected.
	var p string
	v := make([]*ratus.Topic, 0)
	it, err := txn.Get(tableTask, indexID)
	if err != nil {
		return nil, err
	}
	for r := it.Next(); r != nil; r = it.Next() {
		t := r.(*ratus.Task)
		if t.Topic != p {
			p = t.Topic
			v = append(v, &ratus.Topic{Name: t.Topic})
		}
		if len(v) >= offset+limit {
			break
		}
	}

	// Slice the results based on limit and offset.
	n := len(v)
	if n < offset {
		v = v[n:]
	} else if n >= offset {
		v = v[offset:]
	}

	txn.Commit()
	return v, nil
}

// DeleteTopics deletes all topics and tasks.
func (g *Engine) DeleteTopics(ctx context.Context) (*ratus.Deleted, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	n, err := txn.DeleteAll(tableTask, indexID)
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

	// Currently the information of a topic only contains the name and count.
	// Information such as progress and states could be added in the future.
	txn.Commit()
	if n == 0 {
		return nil, ratus.ErrNotFound
	}
	return &ratus.Topic{
		Name:  topic,
		Count: n,
	}, nil
}

// DeleteTopic deletes a topic and its tasks.
func (g *Engine) DeleteTopic(ctx context.Context, topic string) (*ratus.Deleted, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	n, err := txn.DeleteAll(tableTask, indexTopic, topic)
	if err != nil {
		return nil, err
	}

	txn.Commit()
	return &ratus.Deleted{
		Deleted: int64(n),
	}, nil
}
