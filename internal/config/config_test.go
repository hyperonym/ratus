package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/alexflint/go-arg"

	"github.com/hyperonym/ratus/internal/config"
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

func TestServerConfig(t *testing.T) {
	var c config.ServerConfig
	parse(t, "-p 8000 --bind=192.168.1.1", &c)
	if c.Port != 8000 {
		t.Fail()
	}
	if c.Bind != "192.168.1.1" {
		t.Fail()
	}
}

func TestChoreConfig(t *testing.T) {
	var c config.ChoreConfig
	parse(t, "--chore-interval 3m -chore-initial-delay 3500ms --chore-initial-random", &c)
	if c.Interval != 3*time.Minute {
		t.Fail()
	}
	if c.InitialDelay != 3500*time.Millisecond {
		t.Fail()
	}
	if !c.InitialRandom {
		t.Fail()
	}
}

func TestPaginationConfig(t *testing.T) {
	var c config.PaginationConfig
	parse(t, "--pagination-max-limit=15 --pagination-max-offset=99", &c)
	if c.MaxLimit != 15 {
		t.Fail()
	}
	if c.MaxOffset != 99 {
		t.Fail()
	}
}
