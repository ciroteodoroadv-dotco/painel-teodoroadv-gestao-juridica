package calculo

import (
	"testing"
	"time"
)

// D monta uma data civil (UTC, meia-noite).
func D(a int, m time.Month, d int) time.Time { return time.Date(a, m, d, 0, 0, 0, 0, time.UTC) }

// Datas conferidas em calendário litúrgico (valores fixos, não calculados pela função).
func TestPascoa(t *testing.T) {
	casos := map[int]time.Time{2024: D(2024, 3, 31), 2025: D(2025, 4, 20), 2026: D(2026, 4, 5), 2027: D(2027, 3, 28)}
	for ano, esperado := range casos {
		if got := Pascoa(ano); !got.Equal(esperado) {
			t.Errorf("Pascoa(%d) = %v, quer %v", ano, got, esperado)
		}
	}
}
