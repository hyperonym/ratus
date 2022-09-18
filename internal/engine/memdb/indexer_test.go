package memdb_test

import (
	"testing"
	"time"

	"github.com/hyperonym/ratus"
	"github.com/hyperonym/ratus/internal/engine/memdb"
)

func TestIndexer(t *testing.T) {
	type indexer interface {
		FromObject(any) (bool, []byte, error)
		FromArgs(...any) ([]byte, error)
	}
	for _, x := range []struct {
		name    string
		indexer indexer
	}{
		{"state", &memdb.StateFieldIndex{
			Field:  "State",
			Filter: ratus.TaskStatePending,
		}},
		{"time", &memdb.TimeFieldIndex{
			Field: "Scheduled",
		}},
	} {
		p := x
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()

			t.Run("object", func(t *testing.T) {
				t.Parallel()

				t.Run("normal", func(t *testing.T) {
					t.Parallel()
					n := time.Now()
					ok, _, err := p.indexer.FromObject(&struct {
						State     ratus.TaskState
						Scheduled *time.Time
					}{
						ratus.TaskStatePending,
						&n,
					})
					if !ok {
						t.Fail()
					}
					if err != nil {
						t.Error(err)
					}
				})

				t.Run("invalid", func(t *testing.T) {
					t.Parallel()
					ok, _, err := p.indexer.FromObject(&struct{}{})
					if ok {
						t.Fail()
					}
					if err != nil {
						t.Error(err)
					}
				})

				t.Run("type", func(t *testing.T) {
					t.Parallel()
					ok, _, err := p.indexer.FromObject(&struct {
						State     string
						Scheduled string
					}{})
					if ok {
						t.Fail()
					}
					if err == nil {
						t.Fail()
					}
				})

				t.Run("filter", func(t *testing.T) {
					t.Parallel()
					ok, _, err := p.indexer.FromObject(&struct {
						State     ratus.TaskState
						Scheduled *time.Time
					}{
						ratus.TaskStateActive,
						nil,
					})
					if ok {
						t.Fail()
					}
					if err != nil {
						t.Error(err)
					}
				})
			})

			t.Run("args", func(t *testing.T) {
				t.Parallel()

				t.Run("count", func(t *testing.T) {
					t.Parallel()
					if _, err := p.indexer.FromArgs(1, 2); err == nil {
						t.Fail()
					}
				})

				t.Run("invalid", func(t *testing.T) {
					t.Parallel()
					if _, err := p.indexer.FromArgs(nil); err == nil {
						t.Fail()
					}
				})

				t.Run("type", func(t *testing.T) {
					t.Parallel()
					if _, err := p.indexer.FromArgs("foo"); err == nil {
						t.Fail()
					}
				})

				t.Run("filter", func(t *testing.T) {
					t.Parallel()
					if _, err := p.indexer.FromArgs(ratus.TaskStateActive); err == nil {
						t.Fail()
					}
				})
			})
		})
	}
}
