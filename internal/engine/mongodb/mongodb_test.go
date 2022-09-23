package mongodb_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alexflint/go-arg"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/hyperonym/ratus/internal/engine"
	"github.com/hyperonym/ratus/internal/engine/mongodb"
)

const mongoURI = "mongodb://127.0.0.1:27017"

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

func getIndexes(ctx context.Context, t *testing.T, g *mongodb.Engine) []bson.M {
	t.Helper()
	c, err := g.Collection().Indexes().List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var r []bson.M
	if err := c.All(ctx, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func getExpireAfterSeconds(t *testing.T, indexes []bson.M) int32 {
	t.Helper()
	for _, x := range indexes {
		if v, ok := x["expireAfterSeconds"]; ok {
			switch n := v.(type) {
			case int:
				return int32(n)
			case int32:
				return n
			case int64:
				return int32(n)
			}
		}
	}
	return -1
}

func TestConfig(t *testing.T) {
	var c mongodb.Config
	parse(t, "--mongodb-uri=mongodb://user:pass@127.0.0.1:27017/?readPreference=nearest --mongodb-database test --mongodb-retention-period 24h --mongodb-disable-index-creation", &c)
	if c.URI != "mongodb://user:pass@127.0.0.1:27017/?readPreference=nearest" {
		t.Fail()
	}
	if c.Database != "test" {
		t.Fail()
	}
	if c.Collection != "tasks" {
		t.Fail()
	}
	if c.RetentionPeriod != 24*time.Hour {
		t.Fail()
	}
	if !c.DisableIndexCreation {
		t.Fail()
	}
	if c.DisableAutoFallback {
		t.Fail()
	}
	if c.DisableAtomicPoll {
		t.Fail()
	}
}

func TestSuite(t *testing.T) {
	skipShort(t)
	db := "ratus_test_suite"
	col := fmt.Sprintf("test_suite_%d", time.Now().UnixMicro())

	t.Run("preferred", func(t *testing.T) {
		t.Parallel()
		g, err := mongodb.New(&mongodb.Config{
			URI:        mongoURI,
			Database:   db,
			Collection: col + "_preferred",
		})
		if err != nil {
			t.Fatal(err)
		}
		g.Fallback(-1)
		engine.Test(t, g)
	})

	t.Run("fallback", func(t *testing.T) {
		t.Parallel()
		g, err := mongodb.New(&mongodb.Config{
			URI:        mongoURI,
			Database:   db,
			Collection: col + "_fallback",
		})
		if err != nil {
			t.Fatal(err)
		}
		g.Fallback(1)
		engine.Test(t, g)
	})
}

func TestIndex(t *testing.T) {
	skipShort(t)
	db := "ratus_test_index"
	col := fmt.Sprintf("test_index_%d", time.Now().UnixMicro())

	t.Run("none", func(t *testing.T) {
		ctx := context.Background()
		g, err := mongodb.New(&mongodb.Config{
			URI:                  mongoURI,
			Database:             db,
			Collection:           col,
			DisableIndexCreation: true,
			DisableAutoFallback:  true,
			DisableAtomicPoll:    true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := g.Open(ctx); err != nil {
			t.Fatal(err)
		}
		if err := g.Ready(ctx); err != nil {
			t.Fatal(err)
		}
		if n := getIndexes(ctx, t, g); len(n) > 1 {
			t.Errorf("incorrect number of indexes, expected 0 or 1, got %d", len(n))
		}
		if err := g.Close(ctx); err != nil {
			t.Fatal(err)
		}
	})

	// Give the MongoDB server some time to complete background tasks.
	time.Sleep(1 * time.Second)

	t.Run("create", func(t *testing.T) {
		ctx := context.Background()
		g, err := mongodb.New(&mongodb.Config{
			URI:             mongoURI,
			Database:        db,
			Collection:      col,
			RetentionPeriod: 3 * time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := g.Open(ctx); err != nil {
			t.Fatal(err)
		}
		m := getIndexes(ctx, t, g)
		if len(m) != 6 {
			t.Errorf("incorrect number of indexes, expected 6, got %d", len(m))
		}
		if s := getExpireAfterSeconds(t, m); s != 3 {
			t.Errorf("incorrect retention duration, expected 3, got %d", s)
		}
		if err := g.Close(ctx); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("change", func(t *testing.T) {
		ctx := context.Background()
		g, err := mongodb.New(&mongodb.Config{
			URI:             mongoURI,
			Database:        db,
			Collection:      col,
			RetentionPeriod: 7500 * time.Millisecond,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := g.Open(ctx); err != nil {
			t.Fatal(err)
		}
		m := getIndexes(ctx, t, g)
		if len(m) != 6 {
			t.Errorf("incorrect number of indexes, expected 6, got %d", len(m))
		}
		if s := getExpireAfterSeconds(t, m); s != 7 {
			t.Errorf("incorrect retention duration, expected 7, got %d", s)
		}
		if err := g.Destroy(ctx); err != nil {
			t.Fatal(err)
		}
	})
}

func TestError(t *testing.T) {
	t.Run("uri", func(t *testing.T) {
		if _, err := mongodb.New(&mongodb.Config{URI: "invalid"}); err == nil {
			t.Fail()
		}
	})

	t.Run("context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		g, err := mongodb.New(&mongodb.Config{
			URI:      "mongodb://invalid",
			Database: "invalid",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := g.Open(ctx); err == nil {
			t.Fatal(err)
		}
		if err := g.Ready(ctx); err == nil {
			t.Fatal(err)
		}
		if err := g.Destroy(ctx); err == nil {
			t.Fatal(err)
		}
	})
}
