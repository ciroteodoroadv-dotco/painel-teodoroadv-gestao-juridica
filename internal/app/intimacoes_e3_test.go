package app

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/intimacoes/djen"
	"painelteodoro/internal/tarefas"
)

// E3: individual (cada intimação) + consolidada (por processo e dia), ambas com prazo interno = criação + 2 dias (SP).
func TestE3TarefaIndividualEConsolidadaComPrazoInterno(t *testing.T) {
	f := &falsoDJEN{itens: []djen.Comunicacao{comunicacao("1", "0000101-57.2025.8.26.0100"), comunicacao("2", "0000101-57.2025.8.26.0100"), comunicacao("3", "")}}
	e, ac, _ := ambDJEN(t, f)
	adv, _ := e.user("adv-e3@exemplo.test", auth.Advogado)
	adv.criarProcesso(t, "0000101-57.2025.8.26.0100")
	if r, err := e.svc.Intimacoes.ColetarAgora(context.Background(), ac); err != nil || r.Novas != 3 {
		t.Fatalf("coleta = %+v %v", r, err)
	}
	if n := e.count(`SELECT count(*) FROM tarefas WHERE intimacao_id IS NOT NULL AND prazo_interno AND prazo_fatal = DATE '2026-11-05'`); n != 3 {
		t.Errorf("3 tarefas individuais com prazo interno = %d", n)
	}
	if n := e.count(`SELECT count(*) FROM tarefas WHERE consolidada_dia = DATE '2026-11-03' AND prazo_interno AND prazo_fatal = DATE '2026-11-05' AND estado='a-confirmar'`); n != 1 {
		t.Errorf("1 consolidada do dia para o processo cadastrado (a sem processo não tem) = %d", n)
	}
	if n := e.count(`SELECT count(*) FROM prazos WHERE tarefa_id IS NOT NULL AND situacao='a_conferir' AND data_fatal = DATE '2026-11-05'`); n != 4 {
		t.Errorf("4 prazos internos 'a conferir' no domínio Prazos = %d", n)
	}
	var desc string
	_ = e.pool.QueryRow(context.Background(), `SELECT descricao FROM tarefas WHERE consolidada_dia IS NOT NULL`).Scan(&desc)
	if !strings.Contains(desc, tarefas.TextoPrazoInterno) || !strings.Contains(desc, "DJEN") {
		t.Errorf("descrição da consolidada: %q", desc)
	}
	// Segunda coleta no mesmo dia, mesmas comunicações: nada novo.
	if _, err := e.svc.Intimacoes.ColetarAgora(context.Background(), ac); err != nil {
		t.Fatal(err)
	}
	if e.count(`SELECT count(*) FROM tarefas WHERE consolidada_dia IS NOT NULL`) != 1 || e.tarefasDeIntimacao() != 3 {
		t.Error("nada pode ser duplicado")
	}
}

// Decidir (Confirmar/Cancelar) encerra o prazo interno da tarefa individual.
func TestE3DecidirIntimacaoEncerraPrazoInterno(t *testing.T) {
	f := &falsoDJEN{itens: []djen.Comunicacao{comunicacao("1", ""), comunicacao("2", "")}}
	e, ac, _ := ambDJEN(t, f)
	ctx := context.Background()
	if _, err := e.svc.Intimacoes.ColetarAgora(ctx, ac); err != nil {
		t.Fatal(err)
	}
	var i1, i2 string
	_ = e.pool.QueryRow(ctx, `SELECT id::text FROM intimacoes ORDER BY hash LIMIT 1`).Scan(&i1)
	_ = e.pool.QueryRow(ctx, `SELECT id::text FROM intimacoes ORDER BY hash DESC LIMIT 1`).Scan(&i2)
	if err := e.svc.Intimacoes.Decidir(ctx, ac, i1, "confirmada"); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Intimacoes.Cancelar(ctx, ac, i2, "Duplicada"); err != nil {
		t.Fatal(err)
	}
	if e.count(`SELECT count(*) FROM tarefas WHERE intimacao_id IS NOT NULL AND (prazo_interno OR prazo_fatal IS NOT NULL)`) != 0 {
		t.Error("o prazo interno acaba quando a intimação é decidida")
	}
	if e.count(`SELECT count(*) FROM prazos WHERE tarefa_id IS NOT NULL AND situacao='cancelado'`) != 2 {
		t.Error("os dois prazos internos deveriam estar encerrados no domínio Prazos")
	}
}

// E2: texto integral (sem corte de 2000), resumo continua em 2000; a coluna texto guarda tudo.
func TestE2TextoIntegralDaComunicacao(t *testing.T) {
	c := comunicacao("1", "")
	c.Texto = djen.NormalizarTexto(strings.Repeat("palavra € ", 1000))
	f := &falsoDJEN{itens: []djen.Comunicacao{c}}
	e, ac, _ := ambDJEN(t, f)
	if _, err := e.svc.Intimacoes.ColetarAgora(context.Background(), ac); err != nil {
		t.Fatal(err)
	}
	var texto, resumo string
	_ = e.pool.QueryRow(context.Background(), `SELECT texto, resumo FROM intimacoes`).Scan(&texto, &resumo)
	if texto != c.Texto || utf8.RuneCountInString(resumo) != 2000 {
		t.Errorf("texto integral = %d, resumo = %d", utf8.RuneCountInString(texto), utf8.RuneCountInString(resumo))
	}
}
