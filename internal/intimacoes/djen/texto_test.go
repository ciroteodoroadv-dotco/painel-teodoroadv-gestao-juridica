package djen

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNormalizarTextoGuardaOTextoIntegral(t *testing.T) {
	in := "<p>Fica intimada</p> " + strings.Repeat("€ ", 5000) + "fim"
	got := NormalizarTexto(in)
	if !strings.HasPrefix(got, "Fica intimada") || !strings.HasSuffix(got, "fim") || strings.Contains(got, "<p>") {
		t.Errorf("texto integral sem marcação: %.40q ... %.10q", got, got[len(got)-10:])
	}
	if utf8.RuneCountInString(got) < 10000 {
		t.Errorf("não pode cortar em 2000 caracteres: %d", utf8.RuneCountInString(got))
	}
}

func TestNormalizarTextoTetoDeSegurancaMarcaOCorte(t *testing.T) {
	got := NormalizarTexto(strings.Repeat("ç", MaxTexto+500))
	if !strings.HasSuffix(got, MarcaCorte) || !utf8.ValidString(got) {
		t.Errorf("deveria terminar com a marca de corte: %q", got[len(got)-30:])
	}
	if utf8.RuneCountInString(got) != MaxTexto+utf8.RuneCountInString(MarcaCorte) {
		t.Errorf("tamanho = %d", utf8.RuneCountInString(got))
	}
	if MarcaCorte != "[texto cortado]" {
		t.Errorf("marca = %q", MarcaCorte)
	}
	if utf8.RuneCountInString(NormalizarResumo(got)) != MaxResumo {
		t.Error("o resumo continua em 2000 caracteres")
	}
}
