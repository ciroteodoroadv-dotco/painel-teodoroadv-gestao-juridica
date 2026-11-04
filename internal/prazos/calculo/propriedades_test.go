package calculo

import (
	"reflect"
	"testing"
	"time"
)

func ehNaoUtilNacional(d time.Time) bool {
	if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		return true
	}
	_, ok := Nacional{}.NaoUtil(d)
	return ok
}

// Caso 12: propriedades sobre uma varredura de datas, regras e prazos.
func TestPropriedades(t *testing.T) {
	regras := []Regra{CPCUteis, CPPCorridos, MaterialCivel, MaterialPenal}
	for _, regra := range regras {
		for _, modo := range []Modo{DJe, Ciencia} {
			for off := 0; off < 400; off += 3 {
				ev := D(2026, 1, 1).AddDate(0, 0, off)
				var anterior time.Time
				for dias := 1; dias <= 30; dias += 4 {
					e := Entrada{Evento: ev, Modo: modo, Regra: regra, Dias: dias, Hoje: ev}
					r, err := Calcular(e)
					if err != nil {
						t.Fatal(err)
					}
					v := *r.Vencimento
					if r.Inicio != nil && v.Before(*r.Inicio) && regra != MaterialPenal {
						t.Fatalf("%s %s %v +%d: vencimento antes do início", regra, modo, ev, dias)
					}
					if regra == CPCUteis && ehNaoUtilNacional(v) {
						t.Fatalf("CPC %v +%d: vencimento em dia não útil %v", ev, dias, v)
					}
					if regra == CPCUteis && ((v.Month() == time.December && v.Day() >= 20) || (v.Month() == time.January && v.Day() <= 20)) {
						t.Fatalf("CPC %v +%d: vencimento dentro da suspensão %v", ev, dias, v)
					}
					if !anterior.IsZero() && v.Before(anterior) {
						t.Fatalf("%s %s %v: aumentar os dias antecipou o vencimento (%v -> %v)", regra, modo, ev, anterior, v)
					}
					anterior = v
					r2, _ := Calcular(e)
					if !reflect.DeepEqual(r, r2) {
						t.Fatal("mesma entrada, resultados diferentes")
					}
				}
			}
		}
	}
}

// Nenhum dia da semente nacional é excluído sem fonte.
func TestNacionalSempreComFonte(t *testing.T) {
	for d := D(2024, 1, 1); d.Year() < 2030; d = d.AddDate(0, 0, 1) {
		if x, ok := (Nacional{}).NaoUtil(d); ok && x.Fonte == "" {
			t.Errorf("%v excluído sem fonte", d)
		}
	}
}
