package ratelimit

import (
	"testing"
	"time"
)

func TestLimite(t *testing.T) {
	l := New(3, time.Minute)
	now := time.Now()
	l.now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		if !l.Allow("ip") {
			t.Fatalf("tentativa %d deveria passar", i+1)
		}
	}
	if l.Allow("ip") {
		t.Error("4ª tentativa deveria ser barrada")
	}
	if !l.Allow("outro") {
		t.Error("outra chave não deve ser afetada")
	}
	now = now.Add(2 * time.Minute)
	if !l.Allow("ip") {
		t.Error("após a janela deve liberar")
	}
}

func TestAllowMaxTetoProprio(t *testing.T) {
	l := New(2, time.Minute)
	for i := 0; i < 5; i++ {
		if !l.AllowMax("largo", 5) {
			t.Fatalf("tentativa %d do teto largo recusada", i+1)
		}
	}
	if l.AllowMax("largo", 5) {
		t.Fatal("sexta tentativa deveria passar do teto 5")
	}
	primeira, segunda, terceira := l.Allow("estreito"), l.Allow("estreito"), l.Allow("estreito")
	if !primeira || !segunda || terceira {
		t.Fatal("Allow continua com o teto do limitador (2)")
	}
}
