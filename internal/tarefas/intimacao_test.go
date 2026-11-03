package tarefas

import (
	"context"
	"strings"
	"testing"
	"time"

	"painelteodoro/internal/platform/webx"
)

// novaIntimacao grava uma intimação fictícia direto no banco (este pacote não conhece o domínio Intimações).
func (a *amb) novaIntimacao(t *testing.T, hash string) string {
	t.Helper()
	var id string
	if err := a.pool.QueryRow(context.Background(), `INSERT INTO intimacoes (hash, tipo, processo_numero) VALUES ($1,'Intimação','0000101-25.2025.8.26.0100') RETURNING id::text`, hash).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = a.pool.Exec(context.Background(), `TRUNCATE intimacoes, oab_monitoradas, coletas CASCADE`)
	})
	return id
}

func (a *amb) analise(intim, proc string) AnaliseIn {
	return AnaliseIn{IntimacaoID: intim, ProcessoID: proc, Titulo: "Analisar intimação — Intimação — 0000101-25.2025.8.26.0100",
		Descricao: "Prazo a apurar pelo advogado. O sistema não calculou prazo.\nLink: https://exemplo.test/c/1",
		CriadoPor: a.ac.UserID, AuditActor: a.ac.UserID, RequestID: "t"}
}

func TestAnaliseDeIntimacaoNasceAConfirmarSemPrazo(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	intim := a.novaIntimacao(t, "h-analise")
	tx, _ := a.pool.Begin(ctx)
	id, err := a.s.CriarAnaliseDeIntimacao(ctx, tx, a.analise(intim, ""))
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	tt, err := a.s.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if tt.Estado != "a-confirmar" || tt.Prioridade != "Normal" || tt.Subtitulo != "Análise de decisão" {
		t.Errorf("estado/prioridade/subtítulo: %+v", tt)
	}
	if tt.PrazoFatal != nil || tt.Prazo != nil || tt.DataBase != nil {
		t.Errorf("o sistema não presume prazo: prazo=%v fatal=%v base=%v", tt.Prazo, tt.PrazoFatal, tt.DataBase)
	}
	if tt.OrigemTipo != "intimacao" || tt.OrigemID != intim || tt.OrigemCodigo != "DJEN" {
		t.Errorf("origem sem processo deve ser DJEN: tipo=%q id=%q código=%q", tt.OrigemTipo, tt.OrigemID, tt.OrigemCodigo)
	}
	if !strings.Contains(tt.Descricao, "Prazo a apurar pelo advogado") || !strings.HasPrefix(tt.Codigo, "TAR. ") {
		t.Errorf("descrição/código: %+v", tt)
	}
	if len(a.prazos.calls) != 0 {
		t.Errorf("nada pode ser enviado ao domínio Prazos: %+v", a.prazos.calls)
	}
	var n int
	_ = a.pool.QueryRow(ctx, `SELECT count(*) FROM prazos`).Scan(&n)
	if n != 0 {
		t.Errorf("a tarefa de análise não pode virar prazo (%d)", n)
	}
}

func TestAnaliseDeIntimacaoEIdempotenteEVinculaProcesso(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	intim := a.novaIntimacao(t, "h-idem")
	var ids [2]string
	for i := range ids {
		tx, _ := a.pool.Begin(ctx)
		id, err := a.s.CriarAnaliseDeIntimacao(ctx, tx, a.analise(intim, a.pro))
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		ids[i] = id
	}
	if ids[0] != ids[1] {
		t.Errorf("segunda chamada deveria devolver a mesma tarefa: %v", ids)
	}
	var n int
	_ = a.pool.QueryRow(ctx, `SELECT count(*) FROM tarefas WHERE intimacao_id=$1`, intim).Scan(&n)
	if n != 1 {
		t.Errorf("tarefas da intimação = %d", n)
	}
	tt, _ := a.s.Get(ctx, ids[0])
	if tt.OrigemTipo != "processo" || tt.OrigemID != a.pro || !strings.Contains(tt.OrigemCodigo, "PRO. 0000001") || !strings.Contains(tt.OrigemCodigo, "DJEN") {
		t.Errorf("com processo existente o vínculo é o processo (e a origem diz DJEN): %+v", tt)
	}
}

func TestAnaliseDeIntimacaoDesfazComATransacao(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	intim := a.novaIntimacao(t, "h-rollback")
	tx, _ := a.pool.Begin(ctx)
	if _, err := a.s.CriarAnaliseDeIntimacao(ctx, tx, a.analise(intim, "")); err != nil {
		t.Fatal(err)
	}
	_ = tx.Rollback(ctx)
	var n int
	_ = a.pool.QueryRow(ctx, `SELECT count(*) FROM tarefas WHERE intimacao_id=$1`, intim).Scan(&n)
	if n != 0 {
		t.Error("rollback da transação do chamador deve desfazer a tarefa")
	}
}

func TestPorIntimacoesDevolveTituloEEstadoComRotulo(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	i1, i2 := a.novaIntimacao(t, "h-por1"), a.novaIntimacao(t, "h-por2")
	tx, _ := a.pool.Begin(ctx)
	id, err := a.s.CriarAnaliseDeIntimacao(ctx, tx, a.analise(i1, ""))
	if err != nil {
		t.Fatal(err)
	}
	_ = tx.Commit(ctx)
	m, err := a.s.PorIntimacoes(ctx, []string{i1, i2})
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 1 || m[i1].Estado != "A confirmar" || !strings.HasPrefix(m[i1].Titulo, "Analisar intimação") {
		t.Errorf("PorIntimacoes = %+v", m)
	}
	if err := a.s.Mover(ctx, a.ac, id, 1, "pendente", true); err != nil {
		t.Fatal(err)
	}
	m, _ = a.s.PorIntimacoes(ctx, []string{i1})
	if m[i1].Estado != "Pendente" {
		t.Errorf("estado deveria acompanhar a tarefa: %+v", m)
	}
	if m, err := a.s.PorIntimacoes(ctx, nil); err != nil || len(m) != 0 {
		t.Errorf("sem ids: %v %v", m, err)
	}
}

func TestTarefaDeIntimacaoNaListaEmAgendaEDocumentos(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	intim := a.novaIntimacao(t, "h-lista")
	tx, _ := a.pool.Begin(ctx)
	id, err := a.s.CriarAnaliseDeIntimacao(ctx, tx, a.analise(intim, ""))
	if err != nil {
		t.Fatal(err)
	}
	_ = tx.Commit(ctx)
	l, err := a.s.Listar(ctx, "djen")
	if err != nil || len(l) != 1 || l[0].OrigemCodigo != "DJEN" {
		t.Fatalf("busca por DJEN: %d %v", len(l), err)
	}
	linhas, err := a.s.AgendaEmAberto(ctx, webx.Hoje(time.Now()))
	if err != nil || len(linhas) != 1 || linhas[0].StatusRotulo != "A confirmar" {
		t.Errorf("agenda: %+v %v", linhas, err)
	}
	// Sem processo, não há onde guardar o PDF: recusa com mensagem clara em vez de erro interno.
	err = a.s.AnexarDocumento(ctx, a.ac, id, "x.pdf", strings.NewReader("%PDF"))
	invalido(t, err, "Vincule a tarefa a um processo ou atendimento para anexar documentos.")
	if len(a.docs.nomes) != 0 {
		t.Error("nada pode ser enviado ao domínio Documentos")
	}
	// Editar mantém o vínculo e não exige processo.
	in := Input{Titulo: "Analisar intimação (ajustado)", Estado: "pendente", Prioridade: "Alta"}
	if err := a.s.Update(ctx, a.ac, id, 1, in, true); err != nil {
		t.Fatalf("editar tarefa de intimação: %v", err)
	}
	tt, _ := a.s.Get(ctx, id)
	if tt.OrigemTipo != "intimacao" || tt.Titulo != "Analisar intimação (ajustado)" {
		t.Errorf("%+v", tt)
	}
}
