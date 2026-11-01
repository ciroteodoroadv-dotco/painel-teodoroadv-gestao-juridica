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
