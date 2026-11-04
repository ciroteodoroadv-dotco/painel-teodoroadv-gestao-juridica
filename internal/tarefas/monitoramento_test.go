package tarefas

import (
	"context"
	"strings"
	"testing"
)

func (a *amb) garantir(t *testing.T, numero, titulo string) (bool, error) {
	t.Helper()
	ctx := context.Background()
	tx, _ := a.pool.Begin(ctx)
	defer func() { _ = tx.Rollback(ctx) }()
	criada, err := a.s.GarantirMonitoramento(ctx, tx, a.ac, MonitorIn{ProcessoID: a.pro, Numero: numero, Titulo: titulo})
	if err == nil {
		err = tx.Commit(ctx)
	}
	return criada, err
}

func (a *amb) baixar(t *testing.T, motivo string) (bool, error) {
	t.Helper()
	ctx := context.Background()
	tx, _ := a.pool.Begin(ctx)
	defer func() { _ = tx.Rollback(ctx) }()
	ach, err := a.s.BaixarMonitoramento(ctx, tx, a.ac, a.pro, motivo)
	if err == nil {
		err = tx.Commit(ctx)
	}
	return ach, err
}

func (a *amb) monitoramentos(t *testing.T) (abertas, total int) {
	t.Helper()
	_ = a.pool.QueryRow(context.Background(), `SELECT count(*) FILTER (WHERE estado NOT IN ('concluidas-com-sucesso','concluidas-sem-sucesso','canceladas')), count(*) FROM tarefas WHERE monitoramento AND processo_id=$1`, a.pro).Scan(&abertas, &total)
	return
}

func TestMonitoramentoCriaComTextosDoBESemPrazo(t *testing.T) {
	a := novoAmb(t)
	criada, err := a.garantir(t, "0000101-57.2025.8.26.0100", "MARIA x JOÃO")
	if err != nil || !criada {
		t.Fatalf("criada=%v %v", criada, err)
	}
	var titulo, sub, estado, desc, prio string
	var fatal, prazo *string
	_ = a.pool.QueryRow(context.Background(), `SELECT titulo, subtitulo, estado, descricao, prioridade, prazo_fatal::text, prazo::text FROM tarefas WHERE monitoramento`).Scan(&titulo, &sub, &estado, &desc, &prio, &fatal, &prazo)
	if titulo != "Monitoramento processual — 0000101-57.2025.8.26.0100" || sub != "Monitoramento" || estado != "pendente" || prio != "Normal" || fatal != nil || prazo != nil {
		t.Errorf("tarefa: %q %q %q %q fatal=%v prazo=%v", titulo, sub, estado, prio, fatal, prazo)
	}
	if !strings.Contains(desc, "Tarefa criada automaticamente no cadastro do processo") || strings.Contains(desc, "ATENÇÃO") {
		t.Errorf("descrição: %q", desc)
	}
	if len(a.prazos.calls) != 0 {
		t.Errorf("a tarefa de monitoramento não tem prazo: %+v", a.prazos.calls)
	}
}

func TestMonitoramentoSemNumeroAvisaEUsaOTitulo(t *testing.T) {
	a := novoAmb(t)
	if _, err := a.garantir(t, "", "MARIA x JOÃO"); err != nil {
		t.Fatal(err)
	}
	var titulo, desc string
	_ = a.pool.QueryRow(context.Background(), `SELECT titulo, descricao FROM tarefas WHERE monitoramento`).Scan(&titulo, &desc)
	if titulo != "Monitoramento processual — MARIA x JOÃO" || !strings.Contains(desc, "ATENÇÃO: ficha sem número CNJ") {
		t.Errorf("%q / %q", titulo, desc)
	}
}

func TestMonitoramentoIdempotenteBaixaEmCadeiaENovaNaReativacao(t *testing.T) {
	a := novoAmb(t)
	ctx := context.Background()
	for range 2 {
		if _, err := a.garantir(t, "0000101-57.2025.8.26.0100", ""); err != nil {
			t.Fatal(err)
		}
	}
	if ab, tot := a.monitoramentos(t); ab != 1 || tot != 1 {
		t.Fatalf("idempotente: abertas=%d total=%d", ab, tot)
	}
	ach, err := a.baixar(t, "extinto")
	if err != nil || !ach {
		t.Fatalf("baixar: %v %v", ach, err)
	}
	var estado, desc string
	_ = a.pool.QueryRow(ctx, `SELECT estado, descricao FROM tarefas WHERE monitoramento`).Scan(&estado, &desc)
	if estado != "concluidas-com-sucesso" || !strings.Contains(desc, "Monitoramento encerrado automaticamente em ") || !strings.Contains(desc, "processo movido para extinto") {
		t.Errorf("baixa: %q %q", estado, desc)
	}
	if ach, err = a.baixar(t, "extinto"); err != nil || ach {
		t.Errorf("baixar de novo não encontra tarefa aberta e não é erro: %v %v", ach, err)
	}
	if ab, tot := a.monitoramentos(t); ab != 0 || tot != 1 {
		t.Errorf("a baixada continua listada: abertas=%d total=%d", ab, tot)
	}
	if criada, err := a.garantir(t, "0000101-57.2025.8.26.0100", ""); err != nil || !criada {
		t.Fatalf("reativar: %v %v", criada, err)
	}
	if ab, tot := a.monitoramentos(t); ab != 1 || tot != 2 {
		t.Errorf("a baixada é histórico, a nova é outra: abertas=%d total=%d", ab, tot)
	}
	if ach, _ := a.baixar(t, "arquivado"); !ach {
		t.Error("baixa da nova")
	}
}

func TestBaixarSemTarefaNaoEErro(t *testing.T) {
	a := novoAmb(t)
	if ach, err := a.baixar(t, "extinto"); err != nil || ach {
		t.Errorf("sem tarefa: encontrada=%v err=%v", ach, err)
	}
}
