package mongodb_test

import (
	"strings"
	"testing"
	"time"

	"github.com/alexflint/go-arg"

	"github.com/hyperonym/ratus/internal/engine/mongodb"
)

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
