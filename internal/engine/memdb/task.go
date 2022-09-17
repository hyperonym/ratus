package memdb

import (
	"context"

	"github.com/hyperonym/ratus"
)

// ListTasks lists all tasks in a topic.
func (g *Engine) ListTasks(ctx context.Context, topic string, limit, offset int) ([]*ratus.Task, error) {
	txn := g.database.Txn(false)
	defer txn.Abort()

	v := make([]*ratus.Task, 0)
	it, err := txn.Get(tableTask, indexTopic, topic)
	if err != nil {
		return nil, err
	}
	for i, r := 0, it.Next(); i < offset+limit && r != nil; i, r = i+1, it.Next() {
		if i < offset {
			continue
		}
		t := *r.(*ratus.Task)
		v = append(v, &t)
	}

	txn.Commit()
	return v, nil
}

// InsertTasks inserts a batch of tasks while ignoring existing ones.
func (g *Engine) InsertTasks(ctx context.Context, ts []*ratus.Task) (*ratus.Updated, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	var c int64
	for _, t := range ts {
		r, err := txn.First(tableTask, indexID, t.ID)
		if err != nil {
			return nil, err
		}
		if r != nil {
			continue
		}
		if err := txn.Insert(tableTask, t); err != nil {
			return nil, err
		}
		c++
	}

	txn.Commit()
	return &ratus.Updated{
		Created: c,
		Updated: 0,
	}, nil
}

// UpsertTasks inserts or updates a batch of tasks.
func (g *Engine) UpsertTasks(ctx context.Context, ts []*ratus.Task) (*ratus.Updated, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	var c int64
	for _, t := range ts {
		r, err := txn.First(tableTask, indexID, t.ID)
		if err != nil {
			return nil, err
		}
		if err := txn.Insert(tableTask, t); err != nil {
			return nil, err
		}
		if r == nil {
			c++
		}
	}

	txn.Commit()
	return &ratus.Updated{
		Created: c,
		Updated: int64(len(ts)) - c,
	}, nil
}

// DeleteTasks deletes all tasks in a topic.
func (g *Engine) DeleteTasks(ctx context.Context, topic string) (*ratus.Deleted, error) {
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

// GetTask gets a task by its unique ID.
func (g *Engine) GetTask(ctx context.Context, id string) (*ratus.Task, error) {
	txn := g.database.Txn(false)
	defer txn.Abort()

	r, err := txn.First(tableTask, indexID, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ratus.ErrNotFound
	}

	txn.Commit()
	v := *r.(*ratus.Task)
	return &v, nil
}

// InsertTask inserts a new task.
func (g *Engine) InsertTask(ctx context.Context, t *ratus.Task) (*ratus.Updated, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	r, err := txn.First(tableTask, indexID, t.ID)
	if err != nil {
		return nil, err
	}
	if r != nil {
		return nil, ratus.ErrConflict
	}
	if err := txn.Insert(tableTask, t); err != nil {
		return nil, err
	}

	txn.Commit()
	return &ratus.Updated{
		Created: 1,
		Updated: 0,
	}, nil
}

// UpsertTask inserts or updates a task.
func (g *Engine) UpsertTask(ctx context.Context, t *ratus.Task) (*ratus.Updated, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	var u int64
	r, err := txn.First(tableTask, indexID, t.ID)
	if err != nil {
		return nil, err
	}
	if r != nil {
		u = 1
	}
	if err := txn.Insert(tableTask, t); err != nil {
		return nil, err
	}

	txn.Commit()
	return &ratus.Updated{
		Created: 1 - u,
		Updated: u,
	}, nil
}

// DeleteTask deletes a task by its unique ID.
func (g *Engine) DeleteTask(ctx context.Context, id string) (*ratus.Deleted, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	n, err := txn.DeleteAll(tableTask, indexID, id)
	if err != nil {
		return nil, err
	}

	txn.Commit()
	return &ratus.Deleted{
		Deleted: int64(n),
	}, nil
}
