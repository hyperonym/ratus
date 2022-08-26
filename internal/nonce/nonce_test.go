package nonce_test

import (
	"testing"

	"github.com/hyperonym/ratus/internal/nonce"
)

func TestInitialize(t *testing.T) {
	s := nonce.Generate(16)
	if s == "S1PHEigl1PZxPf9g" {
		t.Error("random number generator may not have been properly initialized and returned a deterministic result")
	}
}

func TestGenerate(t *testing.T) {
	for _, n := range []int{16, 32, 128} {
		u := make(map[string]bool)
		for i := 0; i < 100; i++ {
			s := nonce.Generate(n)
			if len(s) != n {
				t.Errorf("incorrect nonce string length %d, expected %d", len(s), n)
			}
			if u[s] {
				t.Errorf("duplicated nonce detected %q", s)
			}
			u[s] = true
		}
	}
}

func BenchmarkGenerate16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nonce.Generate(16)
	}
}

func BenchmarkGenerate32(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nonce.Generate(32)
	}
}

func BenchmarkGenerate128(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nonce.Generate(128)
	}
}
