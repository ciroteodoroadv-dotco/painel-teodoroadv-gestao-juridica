package djen

import (
	"context"
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

// A consulta por número manda o filtro numeroProcesso e exige o número (a pauta inteira da OAB nunca é lida por esta via).
func TestColetarPorNumeroEnviaOFiltro(t *testing.T) {
	e := &espiao{}
	c, _, _ := novo(t, paginado(1, e))
	if _, err := c.ColetarPorNumero(context.Background(), "123456", "SP", "", d1, d3); err == nil {
		t.Error("sem número deveria ser recusado")
	}
	if len(e.urls) != 0 {
		t.Fatal("sem número, nenhuma requisição")
	}
	r, err := c.ColetarPorNumero(context.Background(), "123456", "SP", "0000101-57.2025.8.26.0100", d1, d3)
	if err != nil || len(r.Itens) != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	if !strings.Contains(e.urls[0], "numeroProcesso=0000101-57.2025.8.26.0100") || !strings.Contains(e.urls[0], "numeroOab=123456") {
		t.Errorf("url: %s", e.urls[0])
	}
	e2 := &espiao{}
	c2, _, _ := novo(t, paginado(1, e2))
	_, _ = c2.Coletar(context.Background(), "123456", "SP", d1, d3)
	if strings.Contains(e2.urls[0], "numeroProcesso") {
		t.Errorf("a coleta por OAB não leva o filtro: %s", e2.urls[0])
	}
}
