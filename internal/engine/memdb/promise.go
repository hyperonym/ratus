package memdb

import (
	"context"
	"time"

	"github.com/hyperonym/ratus"
)

// ListPromises lists all promises in a topic.
func (g *Engine) ListPromises(ctx context.Context, topic string, limit, offset int) ([]*ratus.Promise, error) {
	txn := g.database.Txn(false)
	defer txn.Abort()

	// Iterate through the index to return the specified number of results.
	it, err := txn.Get(tableTask, indexActiveTopic, ratus.TaskStateActive, topic)
	if err != nil {
		return nil, err
	}
	v := make([]*ratus.Promise, 0)
	for i, r := 0, it.Next(); i < offset+limit && r != nil; i, r = i+1, it.Next() {
		if i < offset {
			continue
		}
		t := r.(*ratus.Task)
		v = append(v, &ratus.Promise{
			ID:       t.ID,
			Consumer: t.Consumer,
			Deadline: t.Deadline,
		})
	}

	txn.Commit()
	return v, nil
}

// DeletePromises deletes all promises in a topic.
func (g *Engine) DeletePromises(ctx context.Context, topic string) (*ratus.Deleted, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	// Deleting promises is equivalent to setting the states of the active
	// tasks back to "pending" and clearing the nonce fields.
	var d int64
	it, err := txn.Get(tableTask, indexActiveTopic, ratus.TaskStateActive, topic)
	if err != nil {
		return nil, err
	}
	for r := it.Next(); r != nil; r = it.Next() {
		t := r.(*ratus.Task)
		if err := txn.Insert(tableTask, updateOpsRecover(t)); err != nil {
			return nil, err
		}
		d++
	}

	txn.Commit()
	return &ratus.Deleted{
		Deleted: d,
	}, nil
}

// GetPromise gets a promise by the unique ID of its target task.
func (g *Engine) GetPromise(ctx context.Context, id string) (*ratus.Promise, error) {
	txn := g.database.Txn(false)
	defer txn.Abort()

	// A promise in effect is represented in MemDB as fields of an active task.
	r, err := txn.First(tableTask, indexID, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ratus.ErrNotFound
	}
	t := r.(*ratus.Task)
	if t.State != ratus.TaskStateActive {
		return nil, ratus.ErrNotFound
	}

	txn.Commit()
	return &ratus.Promise{
		ID:       t.ID,
		Consumer: t.Consumer,
		Deadline: t.Deadline,
	}, nil
}

// InsertPromise makes a promise to claim and execute a task if it is in pending state.
func (g *Engine) InsertPromise(ctx context.Context, p *ratus.Promise) (*ratus.Task, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	// Check if the target task is in pending state.
	r, err := txn.First(tableTask, indexID, p.ID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ratus.ErrNotFound
	}
	t := r.(*ratus.Task)
	if t.State != ratus.TaskStatePending {
		return nil, ratus.ErrConflict
	}
	u := updateOpsConsume(t, p, time.Now())
	if err := txn.Insert(tableTask, u); err != nil {
		return nil, err
	}

	txn.Commit()
	return clone(u), nil
}

// UpsertPromise makes a promise to claim and execute a task regardless of its current state.
func (g *Engine) UpsertPromise(ctx context.Context, p *ratus.Promise) (*ratus.Task, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	// Check if the target task exists.
	r, err := txn.First(tableTask, indexID, p.ID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ratus.ErrNotFound
	}
	t := r.(*ratus.Task)
	u := updateOpsConsume(t, p, time.Now())
	if err := txn.Insert(tableTask, u); err != nil {
		return nil, err
	}

	txn.Commit()
	return clone(u), nil
}

// DeletePromise deletes a promise by the unique ID of its target task.
func (g *Engine) DeletePromise(ctx context.Context, id string) (*ratus.Deleted, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	// Deleting a promise is equivalent to setting the state of the target task
	// back to "pending" and clearing the nonce field.
	var d int64
	r, err := txn.First(tableTask, indexID, id)
	if err != nil {
		return nil, err
	}
	if r != nil {
		if t := r.(*ratus.Task); t.State == ratus.TaskStateActive {
			if err := txn.Insert(tableTask, updateOpsRecover(t)); err != nil {
				return nil, err
			}
			d++
		}
	}

	txn.Commit()
	return &ratus.Deleted{
		Deleted: d,
	}, nil
}
