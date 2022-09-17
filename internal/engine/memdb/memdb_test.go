package memdb_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alexflint/go-arg"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/engine"
	"github.com/hyperonym/ratus/internal/engine/memdb"
)

func skipShort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testing in short mode")
	}
}

func parse(t *testing.T, cmd string, v any) {
	t.Helper()
	p, err := arg.NewParser(arg.Config{}, v)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Parse(strings.Split(cmd, " ")); err != nil {
		t.Fatal(err)
	}
}

func TestConfig(t *testing.T) {
	var c memdb.Config
	parse(t, "--memdb-retention-period 24h", &c)
	if c.RetentionPeriod != 24*time.Hour {
		t.Fail()
	}
}

func TestSuite(t *testing.T) {
	skipShort(t)

	g, err := memdb.New(&memdb.Config{
		RetentionPeriod: 1 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	engine.Test(t, g)
}

func TestExpire(t *testing.T) {
	skipShort(t)

	ctx := context.Background()
	g, err := memdb.New(&memdb.Config{
		RetentionPeriod: 1 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Ready(ctx); !errors.Is(err, ratus.ErrServiceUnavailable) {
		t.Errorf("incorrect error type, expected %q, got %q", ratus.ErrServiceUnavailable, err)
	}
	if err := g.Open(ctx); err != nil {
		t.Fatal(err)
	}

	n := time.Now()
	n1 := n.Add(1 * time.Second)
	n2 := n.Add(2 * time.Second)
	if _, err := g.InsertTasks(ctx, []*ratus.Task{
		{
			ID:        "1",
			Topic:     "test",
			State:     ratus.TaskStateCompleted,
			Scheduled: &n,
			Consumed:  &n,
		},
		{
			ID:        "2",
			Topic:     "test",
			State:     ratus.TaskStateCompleted,
			Scheduled: &n1,
			Consumed:  &n1,
		},
		{
			ID:        "3",
			Topic:     "test",
			State:     ratus.TaskStateActive,
			Scheduled: &n,
			Consumed:  &n,
			Deadline:  &n2,
		},
	}); err != nil {
		t.Fatal(err)
	}

	if err := g.Chore(ctx); err != nil {
		t.Error(err)
	}
	v, err := g.ListTasks(ctx, "test", 10, 0)
	if err != nil {
		t.Error(err)
	}
	if len(v) != 3 {
		t.Errorf("incorrect number of results, expected 3, got %d", len(v))
	}

	time.Sleep(1 * time.Second)
	if err := g.Chore(ctx); err != nil {
		t.Error(err)
	}
	v, err = g.ListTasks(ctx, "test", 10, 0)
	if err != nil {
		t.Error(err)
	}
	if len(v) != 2 {
		t.Errorf("incorrect number of results, expected 2, got %d", len(v))
	}

	time.Sleep(1 * time.Second)
	if err := g.Chore(ctx); err != nil {
		t.Error(err)
	}
	v, err = g.ListTasks(ctx, "test", 10, 0)
	if err != nil {
		t.Error(err)
	}
	if len(v) != 1 {
		t.Errorf("incorrect number of results, expected 1, got %d", len(v))
	}
}
