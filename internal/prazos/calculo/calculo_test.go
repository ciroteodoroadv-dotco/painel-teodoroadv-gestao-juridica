package calculo

import (
	"strings"
	"testing"
	"time"
)

// calFake é um calendário de tribunal de teste (cada dia com fonte).
type calFake map[time.Time]DiaNaoUtil

func (c calFake) NaoUtil(d time.Time) (DiaNaoUtil, bool) { x, ok := c[d]; return x, ok }
func (c calFake) Cobre(ano int) bool {
	for d := range c {
		if d.Year() == ano {
			return true
		}
	}
	return false
}

func tribunal(dias ...time.Time) calFake {
	c := calFake{}
	for _, d := range dias {
		c[d] = DiaNaoUtil{Data: d, Tipo: "feriado", Descricao: "dia de teste", Fonte: "Portaria de teste"}
	}
	return c
}

func venc(t *testing.T, e Entrada) Resultado {
	t.Helper()
	r, err := Calcular(e)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if r.Vencimento == nil {
		t.Fatal("sem vencimento")
	}
	return r
}

func quer(t *testing.T, nome string, got *time.Time, esperado time.Time) {
	t.Helper()
	if got == nil || !got.Equal(esperado) {
		t.Errorf("%s = %v, quer %s", nome, got, esperado.Format("02/01/2006"))
	}
}

// Caso 1 (CPC arts. 219, 224 §§2º-3º; 2026-11-06 é sexta-feira).
// Disponibilização sex 06/11 -> publicação seg 09/11 -> dia 1 = ter 10/11.
// Dias úteis: 10(1) 11(2) 12(3) 13(4) 16(5) 17(6) 18(7) 19(8) [20/11 sexta = feriado, Lei 14.759/2023]
// 23(9) 24(10) 25(11) 26(12) 27(13) 30(14) 01/12(15). Vencimento: ter 01/12/2026.
func TestDourado1_DJe15Uteis(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2026, 11, 6), Modo: DJe, Regra: CPCUteis, Dias: 15, Hoje: D(2026, 11, 27)})
	quer(t, "publicação", r.Publicacao, D(2026, 11, 9))
	quer(t, "início", r.Inicio, D(2026, 11, 10))
	quer(t, "vencimento", r.Vencimento, D(2026, 12, 1))
	if len(r.Excluidos) != 1 || !r.Excluidos[0].Data.Equal(D(2026, 11, 20)) || r.Excluidos[0].Fonte == "" {
		t.Errorf("excluídos = %+v, quer só 20/11 com fonte", r.Excluidos)
	}
	// Caso 13: de hoje 27/11 (sex) até 01/12: úteis 30/11 e 01/12 = +2; no dia = 0; 03/12 (qui): 02 e 03 = -2.
	if r.DiasUteisAteVencimento != 2 {
		t.Errorf("dias úteis até o vencimento = %d, quer 2", r.DiasUteisAteVencimento)
	}
	if v := venc(t, Entrada{Evento: D(2026, 11, 6), Modo: DJe, Regra: CPCUteis, Dias: 15, Hoje: D(2026, 12, 1)}); v.DiasUteisAteVencimento != 0 {
		t.Errorf("no dia = %d, quer 0", v.DiasUteisAteVencimento)
	}
	if v := venc(t, Entrada{Evento: D(2026, 11, 6), Modo: DJe, Regra: CPCUteis, Dias: 15, Hoje: D(2026, 12, 3)}); v.DiasUteisAteVencimento != -2 {
		t.Errorf("depois = %d, quer -2", v.DiasUteisAteVencimento)
	}
}

// Caso 2 (CPC art. 220: suspensão de 20/12 a 20/01). 18/12/2026 é sexta.
// Publicação seg 21/12 (dentro da suspensão); nada conta até 20/01. Dia 1 = qui 21/01/2027.
// 5 dias úteis: 21(1) 22(2) 25(3) 26(4) 27(5) -> qua 27/01/2027.
func TestDourado2_Recesso(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2026, 12, 18), Modo: DJe, Regra: CPCUteis, Dias: 5, Hoje: D(2026, 12, 18)})
	quer(t, "início", r.Inicio, D(2027, 1, 21))
	quer(t, "vencimento", r.Vencimento, D(2027, 1, 27))
	if !strings.Contains(strings.Join(r.Premissas, "|"), "20/12") {
		t.Errorf("premissas sem o recesso: %v", r.Premissas)
	}
}

// Caso 3 (CPC art. 224 §1º; Sexta-feira Santa 2027 = 26/03, Páscoa 28/03/2027).
// Ciência qui 25/03 -> o dia seguinte (sex 26/03) é feriado, 27 e 28 fim de semana -> dia 1 = seg 29/03; 1 dia útil -> 29/03/2027.
func TestDourado3_SextaSanta(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2027, 3, 25), Modo: Ciencia, Regra: CPCUteis, Dias: 1, Hoje: D(2027, 3, 25)})
	quer(t, "vencimento", r.Vencimento, D(2027, 3, 29))
	if len(r.Excluidos) != 1 || !r.Excluidos[0].Data.Equal(D(2027, 3, 26)) {
		t.Errorf("excluídos = %+v, quer Sexta-feira Santa 26/03", r.Excluidos)
	}
}

// Caso 4 (CPC art. 224 §3º). Ciência sex 06/11/2026 -> início seg 09/11; 3 dias úteis: 09, 10, 11 -> 11/11/2026.
func TestDourado4_CienciaSexta(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2026, 11, 6), Modo: Ciencia, Regra: CPCUteis, Dias: 3, Hoje: D(2026, 11, 6)})
	quer(t, "início", r.Inicio, D(2026, 11, 9))
	quer(t, "vencimento", r.Vencimento, D(2026, 11, 11))
}

// Caso 5 (CPP art. 798, dias corridos, exclui o dia do começo). Ciência qui 05/11/2026 + 3 dias = dom 08/11 -> prorroga seg 09/11.
// Recesso NÃO aplicado e a dúvida é declarada.
func TestDourado5_CPPDomingo(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2026, 11, 5), Modo: Ciencia, Regra: CPPCorridos, Dias: 3, Hoje: D(2026, 11, 5)})
	quer(t, "vencimento", r.Vencimento, D(2026, 11, 9))
	if !strings.Contains(strings.Join(r.Duvidas, "|"), "[VERIFICAR]") {
		t.Errorf("sem [VERIFICAR] nas dúvidas: %v", r.Duvidas)
	}
}

// CPP, DJe: disponibilização sex 06/11/2026, 5 dias. Publicação (seg 09/11): +5 = sáb 14/11 (sábado não é prorrogado: art. 798 §3º cita domingo/feriado).
// Contando da própria disponibilização: +5 = qua 11/11. O MAIS CURTO (11/11) é o exibido; 14/11 vai nas alternativas.
func TestDourado5b_CPPDJeMostraOMaisCurto(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2026, 11, 6), Modo: DJe, Regra: CPPCorridos, Dias: 5, Hoje: D(2026, 11, 6)})
	quer(t, "vencimento", r.Vencimento, D(2026, 11, 11))
	if !strings.Contains(strings.Join(r.Alternativas, "|"), "14/11/2026") {
		t.Errorf("alternativas = %v, quer citar 14/11/2026", r.Alternativas)
	}
}

// Caso 6a (CC art. 132): exclui o dia do começo, inclui o do vencimento, prorroga se cair em domingo/feriado.
// Evento sex 06/11/2026 + 2 = dom 08/11 -> seg 09/11.  Evento sex 13/11 + 7 = sex 20/11 (feriado) -> seg 23/11.
// Sábado não é prorrogado (dúvida): evento qua 04/11 + 3 = sáb 07/11 (exibido), alternativa seg 09/11.
func TestDourado6a_MaterialCivel(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2026, 11, 6), Modo: Ciencia, Regra: MaterialCivel, Dias: 2, Hoje: D(2026, 11, 6)})
	quer(t, "domingo", r.Vencimento, D(2026, 11, 9))
	r = venc(t, Entrada{Evento: D(2026, 11, 13), Modo: Ciencia, Regra: MaterialCivel, Dias: 7, Hoje: D(2026, 11, 13)})
	quer(t, "feriado", r.Vencimento, D(2026, 11, 23))
	r = venc(t, Entrada{Evento: D(2026, 11, 4), Modo: Ciencia, Regra: MaterialCivel, Dias: 3, Hoje: D(2026, 11, 4)})
	quer(t, "sábado", r.Vencimento, D(2026, 11, 7))
	if !strings.Contains(strings.Join(r.Alternativas, "|"), "09/11/2026") {
		t.Errorf("alternativas = %v, quer citar 09/11/2026", r.Alternativas)
	}
}

// Caso 6b (CP art. 10): INCLUI o dia do começo e NÃO prorroga.
// Evento sex 06/11/2026, 5 dias -> 06,07,08,09,10 -> ter 10/11.  Evento qui 05/11, 4 dias -> 05,06,07,08 -> dom 08/11 (sem prorrogar).
func TestDourado6b_MaterialPenal(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2026, 11, 6), Modo: Ciencia, Regra: MaterialPenal, Dias: 5, Hoje: D(2026, 11, 6)})
	quer(t, "5 dias", r.Vencimento, D(2026, 11, 10))
	r = venc(t, Entrada{Evento: D(2026, 11, 5), Modo: Ciencia, Regra: MaterialPenal, Dias: 4, Hoje: D(2026, 11, 5)})
	quer(t, "domingo", r.Vencimento, D(2026, 11, 8))
}

// Emenda de 04/11/2026: sem prazo em dobro; o resultado diz que o painel não o aplica.
func TestSemDobro_ListadoComoNaoConsiderado(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2026, 11, 6), Modo: Ciencia, Regra: CPCUteis, Dias: 15, Hoje: D(2026, 11, 6)})
	if !strings.Contains(strings.Join(r.NaoConsiderado, "|"), "prazo em dobro (CPC arts. 180, 183, 186 e 229): o painel não o aplica; se o caso o exigir, o advogado ajusta manualmente") {
		t.Errorf("faltou o item de dobro: %v", r.NaoConsiderado)
	}
}

// Caso 8 (Lei 14.759/2023, publicada em 22/12/2023): 20/11/2023 NÃO é feriado; 20/11/2024 é.
// 2023: ciência sex 17/11 -> seg 20/11 é útil -> vence 20/11/2023.
// 2024: ciência ter 19/11 -> qua 20/11 é feriado -> qui 21/11/2024.
func TestDourado8_Consciencia20Nov(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2023, 11, 17), Modo: Ciencia, Regra: CPCUteis, Dias: 1, Hoje: D(2023, 11, 17)})
	quer(t, "2023", r.Vencimento, D(2023, 11, 20))
	r = venc(t, Entrada{Evento: D(2024, 11, 19), Modo: Ciencia, Regra: CPCUteis, Dias: 1, Hoje: D(2024, 11, 19)})
	quer(t, "2024", r.Vencimento, D(2024, 11, 21))
}

// Caso 9 (regressão do bug do B): Quarta-feira de Cinzas (qua 10/02/2027) e Corpus Christi (qui 27/05/2027)
// NÃO são excluídos no calendário nacional; SÃO quando o calendário do tribunal os traz com fonte.
func TestDourado9_CinzasECorpusSoComFonte(t *testing.T) {
	cinzas := Entrada{Evento: D(2027, 2, 9), Modo: Ciencia, Regra: CPCUteis, Dias: 1, Hoje: D(2027, 2, 9)}
	quer(t, "cinzas nacional", venc(t, cinzas).Vencimento, D(2027, 2, 10))
	cinzas.Cal = tribunal(D(2027, 2, 10))
	quer(t, "cinzas tribunal", venc(t, cinzas).Vencimento, D(2027, 2, 11))
	corpus := Entrada{Evento: D(2027, 5, 26), Modo: Ciencia, Regra: CPCUteis, Dias: 1, Hoje: D(2027, 5, 26)}
	quer(t, "corpus nacional", venc(t, corpus).Vencimento, D(2027, 5, 27))
	corpus.Cal = tribunal(D(2027, 5, 27))
	quer(t, "corpus tribunal", venc(t, corpus).Vencimento, D(2027, 5, 28))
}

// Sem calendário do tribunal (ou sem dias no ano): premissa e dúvida dizem isso.
func TestSemCalendarioVerificado(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2026, 11, 6), Modo: Ciencia, Regra: CPCUteis, Dias: 3, Hoje: D(2026, 11, 6)})
	if !strings.Contains(strings.Join(r.Premissas, "|"), "Sem calendário verificado deste tribunal: considerados só os feriados nacionais") || len(r.Duvidas) == 0 {
		t.Errorf("faltou o aviso de calendário: %v / %v", r.Premissas, r.Duvidas)
	}
	e := Entrada{Evento: D(2026, 11, 6), Modo: Ciencia, Regra: CPCUteis, Dias: 3, Hoje: D(2026, 11, 6), Cal: tribunal(D(2025, 3, 3))}
	if r := venc(t, e); !strings.Contains(strings.Join(r.Premissas, "|"), "Sem calendário verificado") {
		t.Errorf("calendário sem dias no ano deveria avisar: %v", r.Premissas)
	}
	e.Cal = tribunal(D(2026, 3, 3))
	if r := venc(t, e); strings.Contains(strings.Join(r.Premissas, "|"), "Sem calendário verificado") {
		t.Errorf("calendário com dias no ano não deveria avisar: %v", r.Premissas)
	}
}

// Caso 11: entradas inválidas.
func TestEntradasInvalidas(t *testing.T) {
	ok := Entrada{Evento: D(2026, 11, 6), Modo: DJe, Regra: CPCUteis, Dias: 5, Hoje: D(2026, 11, 6)}
	casos := []struct {
		nome string
		mut  func(*Entrada)
		msg  string
	}{
		{"dias zero", func(e *Entrada) { e.Dias = 0 }, "Número de dias inválido."},
		{"dias negativo", func(e *Entrada) { e.Dias = -3 }, "Número de dias inválido."},
		{"dias enorme", func(e *Entrada) { e.Dias = MaxDias + 1 }, "Número de dias inválido."},
		{"data zero", func(e *Entrada) { e.Evento = time.Time{} }, "Data do evento inválida."},
		{"regra", func(e *Entrada) { e.Regra = "xyz" }, "Regra do prazo desconhecida."},
		{"modo", func(e *Entrada) { e.Modo = "xyz" }, "Modo de contagem desconhecido."},
	}
	for _, c := range casos {
		e := ok
		c.mut(&e)
		if _, err := Calcular(e); err == nil || err.Error() != c.msg {
			t.Errorf("%s: erro = %v, quer %q", c.nome, err, c.msg)
		}
	}
}
