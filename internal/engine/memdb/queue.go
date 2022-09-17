package memdb

import (
	"context"
	"time"

	"github.com/hyperonym/ratus"
)

// Chore recovers timed out tasks and deletes expired tasks.
func (g *Engine) Chore(ctx context.Context) error {
	txn := g.database.Txn(true)
	defer txn.Abort()

	// Recover tasks that have timed out.
	n := time.Now()
	it, err := txn.LowerBound(tableTask, indexActiveDeadline, ratus.TaskStateActive, time.UnixMilli(0))
	if err != nil {
		return err
	}
	for r := it.Next(); r != nil; r = it.Next() {
		t := r.(*ratus.Task)
		if t.Deadline != nil && t.Deadline.After(n) {
			break
		}
		u := updateOpsRecover(t)
		if err := txn.Insert(tableTask, u); err != nil {
			return err
		}
	}

	// Delete completed tasks that have exceeded their retention period.
	it, err = txn.LowerBound(tableTask, indexCompletedConsumed, ratus.TaskStateCompleted, time.UnixMilli(0))
	if err != nil {
		return err
	}
	for r := it.Next(); r != nil; r = it.Next() {
		t := r.(*ratus.Task)
		if t.Consumed != nil && t.Consumed.Add(g.config.RetentionPeriod).After(n) {
			break
		}
		if err := txn.Delete(tableTask, t); err != nil {
			return err
		}
	}

	txn.Commit()
	return nil
}

// Poll makes a promise to claim and execute the next available task in a topic.
func (g *Engine) Poll(ctx context.Context, topic string, p *ratus.Promise) (*ratus.Task, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	it, err := txn.LowerBound(tableTask, indexPendingTopicScheduled, ratus.TaskStatePending, topic, time.UnixMilli(0))
	if err != nil {
		return nil, err
	}
	r := it.Next()
	if r == nil {
		return nil, ratus.ErrNotFound
	}
	t := r.(*ratus.Task)
	if t.Scheduled != nil && t.Scheduled.After(time.Now()) {
		return nil, ratus.ErrNotFound
	}
	u := updateOpsConsume(t, p, time.Now())
	if err := txn.Insert(tableTask, u); err != nil {
		return nil, err
	}

	txn.Commit()
	v := *u
	return &v, nil
}

// Commit applies a set of updates to a task and returns the updated task.
func (g *Engine) Commit(ctx context.Context, id string, m *ratus.Commit) (*ratus.Task, error) {
	txn := g.database.Txn(true)
	defer txn.Abort()

	r, err := txn.First(tableTask, indexID, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ratus.ErrNotFound
	}
	t := r.(*ratus.Task)
	if m.Nonce != "" && m.Nonce != t.Nonce {
		return nil, ratus.ErrConflict
	}
	u := updateOpsCommit(t, m)
	if err := txn.Insert(tableTask, u); err != nil {
		return nil, err
	}

	txn.Commit()
	v := *u
	return &v, nil
}
