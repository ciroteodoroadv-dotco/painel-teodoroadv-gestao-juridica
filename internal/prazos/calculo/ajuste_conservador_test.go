package calculo

import (
	"strings"
	"testing"
)

func juntar(l []string) string { return strings.Join(l, "|") }

// Ajuste conservador 04/11, ponto 1: sábado coerente. 2026-11-20 é sexta e feriado (Lei 14.759/2023).
// CPP, ciência qui 19/11 + 1 = sex 20/11 (feriado). Prorroga para o dia seguinte, sáb 21/11, e AQUI PARA (sábado não
// é prorrogado, como no vencimento direto em sábado): exibido sáb 21/11. Alternativa (sábado sem expediente): seg 23/11.
// Antes do ajuste o código pulava o sábado só depois da 1ª prorrogação e exibia 23/11.
func TestConservador1_CPPSabadoDepoisDaPrimeiraProrrogacao(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2026, 11, 19), Modo: Ciencia, Regra: CPPCorridos, Dias: 1, Hoje: D(2026, 11, 19)})
	quer(t, "vencimento", r.Vencimento, D(2026, 11, 21))
	if !strings.Contains(juntar(r.Alternativas), "23/11/2026") || !strings.Contains(juntar(r.Duvidas), "[VERIFICAR] vencimento em sábado") {
		t.Errorf("alternativas = %v / dúvidas = %v", r.Alternativas, r.Duvidas)
	}
}

// Ponto 2: material cível com domingo. Evento sex 06/11/2026 + 2 = dom 08/11 (exibido, sem prorrogar).
// Alternativa (domingo tido como feriado): seg 09/11. Dúvida [VERIFICAR] sobre o domingo.
func TestConservador2_CivelDomingoNaoProrroga(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2026, 11, 6), Modo: Ciencia, Regra: MaterialCivel, Dias: 2, Hoje: D(2026, 11, 6)})
	quer(t, "vencimento", r.Vencimento, D(2026, 11, 8))
	if !strings.Contains(juntar(r.Alternativas), "domingo") || !strings.Contains(juntar(r.Alternativas), "09/11/2026") {
		t.Errorf("alternativas = %v", r.Alternativas)
	}
	if !strings.Contains(juntar(r.Duvidas), "[VERIFICAR] prorrogação por domingo") {
		t.Errorf("dúvidas = %v", r.Duvidas)
	}
}

// Ponto 2, feriado continua prorrogando: evento qua 18/11 + 2 = sex 20/11 (feriado) -> sáb 21/11 (exibido);
// alternativa (sábado sem expediente): seg 23/11.
func TestConservador2_CivelFeriadoProrroga(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2026, 11, 18), Modo: Ciencia, Regra: MaterialCivel, Dias: 2, Hoje: D(2026, 11, 18)})
	quer(t, "vencimento", r.Vencimento, D(2026, 11, 21))
	if !strings.Contains(juntar(r.Alternativas), "23/11/2026") {
		t.Errorf("alternativas = %v", r.Alternativas)
	}
	if len(r.Excluidos) != 1 || !r.Excluidos[0].Data.Equal(D(2026, 11, 20)) {
		t.Errorf("excluídos = %+v, quer 20/11", r.Excluidos)
	}
}

// Ponto 3: Sexta-Feira Santa 2026 = 03/04 (Páscoa 05/04/2026; 01/04/2026 é quarta). CPC, ciência qua 01/04 + 3 dias úteis.
// Sem excluir: qui 02(1) sex 03(2) seg 06(3) -> seg 06/04/2026 (exibido, mais curto).
// Excluindo a Sexta Santa: qui 02(1) seg 06(2) ter 07(3) -> alternativa ter 07/04/2026, com [VERIFICAR].
func TestConservador3_SextaSantaNaoExcluidaPorPadrao(t *testing.T) {
	e := Entrada{Evento: D(2026, 4, 1), Modo: Ciencia, Regra: CPCUteis, Dias: 3, Hoje: D(2026, 4, 1)}
	r := venc(t, e)
	quer(t, "vencimento", r.Vencimento, D(2026, 4, 6))
	if len(r.Excluidos) != 0 {
		t.Errorf("excluídos = %+v, quer nenhum", r.Excluidos)
	}
	if !strings.Contains(juntar(r.Alternativas), "Se o tribunal adotar a Sexta-Feira Santa como dia sem expediente, o vencimento seria 07/04/2026") {
		t.Errorf("alternativas = %v", r.Alternativas)
	}
	if !strings.Contains(juntar(r.Duvidas), "[VERIFICAR] Sexta-Feira Santa") {
		t.Errorf("dúvidas = %v", r.Duvidas)
	}
}

// Ponto 3: calendário de tribunal que traz a Sexta Santa COM fonte continua a excluí-la: vencimento ter 07/04/2026
// (mesma conta acima), dia excluído com a fonte do tribunal e nenhuma alternativa por causa da Sexta Santa.
func TestConservador3_SextaSantaComFonteDoTribunalExclui(t *testing.T) {
	e := Entrada{Evento: D(2026, 4, 1), Modo: Ciencia, Regra: CPCUteis, Dias: 3, Hoje: D(2026, 4, 1), Cal: tribunal(D(2026, 4, 3))}
	r := venc(t, e)
	quer(t, "vencimento", r.Vencimento, D(2026, 4, 7))
	if len(r.Excluidos) != 1 || !r.Excluidos[0].Data.Equal(D(2026, 4, 3)) || r.Excluidos[0].Fonte == "" {
		t.Errorf("excluídos = %+v", r.Excluidos)
	}
	if strings.Contains(juntar(r.Alternativas), "Sexta-Feira Santa") {
		t.Errorf("alternativas = %v, não deveria citar a Sexta Santa", r.Alternativas)
	}
}

// Ponto 3 em prazo corrido: CPP, ciência qua 01/04/2026 + 2 = sex 03/04 (Sexta Santa) -> exibido 03/04.
// Se for dia sem expediente: prorroga para sáb 04/04 (sábado fica, ponto 1) -> alternativa 04/04/2026.
func TestConservador3_SextaSantaEmPrazoCorrido(t *testing.T) {
	r := venc(t, Entrada{Evento: D(2026, 4, 1), Modo: Ciencia, Regra: CPPCorridos, Dias: 2, Hoje: D(2026, 4, 1)})
	quer(t, "vencimento", r.Vencimento, D(2026, 4, 3))
	if !strings.Contains(juntar(r.Alternativas), "04/04/2026") {
		t.Errorf("alternativas = %v", r.Alternativas)
	}
}

// A semente nacional não traz mais a Sexta-Feira Santa, e o 12/10 cita a Lei 6.802/1980.
func TestNacionalNaoTemSextaSanta(t *testing.T) {
	if _, ok := (Nacional{}).NaoUtil(D(2026, 4, 3)); ok {
		t.Error("Sexta Santa não deve estar no calendário nacional")
	}
	if x, ok := (Nacional{}).NaoUtil(D(2026, 10, 12)); !ok || !strings.Contains(x.Fonte, "Lei 6.802/1980") {
		t.Errorf("12/10 = %+v, quer fonte Lei 6.802/1980", x)
	}
}
