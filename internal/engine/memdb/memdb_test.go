package memdb_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
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

func exists(t *testing.T, path string) bool {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false
		}
		panic(err)
	}
	return true
}

func TestConfig(t *testing.T) {
	var c memdb.Config
	parse(t, "--memdb-snapshot-path test.db --memdb-snapshot-interval 20s --memdb-retention-period 24h", &c)
	if c.SnapshotPath != "test.db" {
		t.Fail()
	}
	if c.SnapshotInterval != 20*time.Second {
		t.Fail()
	}
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

func TestSnapshot(t *testing.T) {
	skipShort(t)
	ctx := context.Background()
	p := "test.db"
	os.Remove(p)
	g, err := memdb.New(&memdb.Config{
		SnapshotPath:     p,
		SnapshotInterval: 5 * time.Minute,
		RetentionPeriod:  10 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Open(ctx); err != nil {
		t.Fatal(err)
	}

	n := time.Now()
	if _, err := g.InsertTasks(ctx, []*ratus.Task{
		{
			ID:        "1",
			Topic:     "test",
			State:     ratus.TaskStatePending,
			Scheduled: &n,
			Consumed:  &n,
		},
		{
			ID:        "2",
			Topic:     "test",
			State:     ratus.TaskStatePending,
			Scheduled: &n,
			Consumed:  &n,
		},
	}); err != nil {
		t.Fatal(err)
	}

	t.Run("chore", func(t *testing.T) {
		if err := g.Chore(ctx); err != nil {
			t.Error(err)
		}
		if !exists(t, p) {
			t.Fail()
		}
		if err := g.Chore(ctx); err != nil {
			t.Error(err)
		}
	})

	t.Run("close", func(t *testing.T) {
		if _, err := g.InsertTask(ctx, &ratus.Task{
			ID:        "3",
			Topic:     "test",
			State:     ratus.TaskStatePending,
			Scheduled: &n,
			Consumed:  &n,
		}); err != nil {
			t.Error(err)
		}
		if err := g.Close(ctx); err != nil {
			t.Error(err)
		}
		if !exists(t, p) {
			t.Fail()
		}
	})

	t.Run("recover", func(t *testing.T) {
		u, err := memdb.New(&memdb.Config{
			SnapshotPath:     p,
			SnapshotInterval: 5 * time.Minute,
			RetentionPeriod:  10 * time.Minute,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := u.Open(ctx); err != nil {
			t.Fatal(err)
		}
		v, err := u.ListTasks(ctx, "test", 10, 0)
		if err != nil {
			t.Error(err)
		}
		if len(v) != 3 {
			t.Errorf("incorrect number of results, expected %d, got %d", 3, len(v))
		}
		if err := u.Destroy(ctx); err != nil {
			t.Error(err)
		}
		if exists(t, p) {
			t.Fail()
		}
	})
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

	for i := 0; i < 3; i++ {
		if i > 0 {
			time.Sleep(1 * time.Second)
		}
		if err := g.Chore(ctx); err != nil {
			t.Error(err)
		}
		v, err := g.ListTasks(ctx, "test", 10, 0)
		if err != nil {
			t.Error(err)
		}
		if len(v) != 3-i {
			t.Errorf("incorrect number of results, expected %d, got %d", 3-i, len(v))
		}
	}
}
