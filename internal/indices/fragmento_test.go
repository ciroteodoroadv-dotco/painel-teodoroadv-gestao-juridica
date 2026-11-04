package indices

import (
	"strings"
	"testing"
	"time"

	"painelteodoro/internal/platform/coletas"
)

var agora = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)

func data(s string) time.Time { t, _ := time.Parse("2006-01-02", s); return t }

func valores() []Valor {
	return []Valor{
		{Codigo: 432, Rotulo: "Selic (meta Copom)", Unidade: "% a.a.", Valor: "14.900000", Referencia: data("2026-10-29")},
		{Codigo: 433, Rotulo: "IPCA", Unidade: "% mensal", Valor: "0.520000", Referencia: data("2026-10-01")},
		{Codigo: 188, Rotulo: "INPC", Unidade: "% mensal", Valor: "0.40", Referencia: data("2026-10-01")},
		{Codigo: 189, Rotulo: "IGP-M", Unidade: "% mensal", Valor: "-0.10", Referencia: data("2026-10-01")},
		{Codigo: 1, Rotulo: "Dólar (PTAX)", Unidade: "R$", Valor: "5.4321", Referencia: data("2026-11-03")},
		{Codigo: 1619, Rotulo: "Salário mínimo", Unidade: "R$", Valor: "1518.000000", Referencia: data("2026-10-01")},
	}
}

func TestFormatarValoresPtBR(t *testing.T) {
	casos := []struct{ valor, un, esperado string }{
		{"0.520000", "% mensal", "0,52 %"}, {"14.900000", "% a.a.", "14,90 %"}, {"-0.10", "% mensal", "-0,10 %"},
		{"5.4321", "R$", "R$ 5,4321"}, {"5.430000", "R$", "R$ 5,43"}, {"1518.000000", "R$", "R$ 1.518,00"}, {"1234567.5", "R$", "R$ 1.234.567,50"},
	}
	for _, c := range casos {
		if g := Formatar(c.valor, c.un); g != c.esperado {
			t.Errorf("Formatar(%q,%q) = %q, esperado %q", c.valor, c.un, g, c.esperado)
		}
	}
}

func TestTickerComDadosNaOrdemDoB(t *testing.T) {
	e := coletas.Estado{Modulo: Modulo, Status: "sucesso"}
	h := string(Ticker(Montar(valores(), e, true, agora, 40)))
	ordem := []string{"Selic (meta Copom)", "IPCA", "INPC", "IGP-M", "Dólar (PTAX)", "Salário mínimo"}
	pos := -1
	for _, r := range ordem {
		i := strings.Index(h, r)
		if i <= pos {
			t.Fatalf("ordem errada em %q: %s", r, h)
		}
		pos = i
	}
	for _, trecho := range []string{`<b>14,90 %</b>`, `<b>R$ 1.518,00</b>`, `Ref.: 29/10/2026 · sincronizado`, `class="idxt-status"`, `class="idxt-item"`} {
		if !strings.Contains(h, trecho) {
			t.Errorf("faltou %q em %s", trecho, h)
		}
	}
	if strings.Contains(h, "desatualizado") {
		t.Errorf("nada deveria estar desatualizado: %s", h)
	}
	if strings.Count(h, "<b>") != strings.Count(h, `title="Ref. `) {
		t.Errorf("todo valor precisa da data de referência no title: %s", h)
	}
}

func TestTickerSemDadosDizOEstado(t *testing.T) {
	h := string(Ticker(Montar(nil, coletas.Estado{}, false, agora, 40)))
	if !strings.Contains(h, "Índices econômicos: não sincronizado") || strings.Contains(h, "<b>") {
		t.Errorf("sem dados: %s", h)
	}
	e := coletas.Estado{Modulo: Modulo, Status: "bloqueado", Mensagem: "Coleta desligada neste ambiente."}
	h = string(Ticker(Montar(nil, e, true, agora, 40)))
	if !strings.Contains(h, "Índices econômicos: bloqueado — Coleta desligada neste ambiente.") {
		t.Errorf("bloqueado: %s", h)
	}
}

func TestTickerMarcaDesatualizado(t *testing.T) {
	vs := valores()
	vs[1].Referencia = data("2026-09-20") // 45 dias
	vs[2].Falhou = true                   // falhou na última coleta
	h := string(Ticker(Montar(vs, coletas.Estado{Status: "parcial"}, true, agora, 40)))
	if strings.Count(h, `class="idxt-velho"`) != 2 || !strings.Contains(h, "· parcial · desatualizado") {
		t.Errorf("marca de desatualizado: %s", h)
	}
}

func TestTickerSemHandlersNemEstilo(t *testing.T) {
	h := strings.ToLower(string(Ticker(Montar(valores(), coletas.Estado{Status: "sucesso"}, true, agora, 40))))
	for _, ruim := range []string{" on", "style=", "<script"} {
		if strings.Contains(h, ruim) {
			t.Errorf("contém %q: %s", ruim, h)
		}
	}
}
