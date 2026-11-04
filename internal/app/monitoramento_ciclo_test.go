package app

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/webx"
	"painelteodoro/internal/processos"
)

func (e *env) moverProcesso(b *browser, id, novo, motivo string) int {
	e.t.Helper()
	var v int
	_ = e.pool.QueryRow(context.Background(), `SELECT version FROM processos WHERE id=$1::uuid`, id).Scan(&v)
	r, _ := b.post("/processos/"+id+"/estado", url.Values{"version": {strconv.Itoa(v)}, "novo_estado": {novo}, "motivo": {motivo}})
	return r.StatusCode
}

func (e *env) monitor(id string) (abertas, total int) {
	e.t.Helper()
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FILTER (WHERE estado NOT IN ('concluidas-com-sucesso','concluidas-sem-sucesso','canceladas')), count(*)
		FROM tarefas WHERE monitoramento AND processo_id=$1::uuid`, id).Scan(&abertas, &total)
	return
}

func TestMonitoramentoNoCicloDeVidaDoProcesso(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv-cic@exemplo.test", auth.Advogado)
	id := adv.criarProcesso(t, numA)
	if ab, tot := e.monitor(id); ab != 1 || tot != 1 {
		t.Fatalf("processo ativo nasce com 1 tarefa de monitoramento: %d/%d", ab, tot)
	}
	if e.moverProcesso(adv, id, "sobrestado", "") != 200 {
		t.Fatal("mover para sobrestado")
	}
	if ab, tot := e.monitor(id); ab != 1 || tot != 1 {
		t.Errorf("ativo→sobrestado mantém a mesma (idempotente): %d/%d", ab, tot)
	}
	if e.moverProcesso(adv, id, "extinto", "Sentença transitada") != 200 {
		t.Fatal("mover para extinto")
	}
	var estado, desc string
	_ = e.pool.QueryRow(context.Background(), `SELECT estado, descricao FROM tarefas WHERE monitoramento AND processo_id=$1::uuid`, id).Scan(&estado, &desc)
	if ab, tot := e.monitor(id); ab != 0 || tot != 1 || estado != "concluidas-com-sucesso" || !strings.Contains(desc, "processo movido para extinto") {
		t.Errorf("ativo→extinto baixa a tarefa e ela continua listada: %d/%d %q", ab, tot, estado)
	}
	if e.count(`SELECT count(*) FROM processo_andamentos WHERE processo_id=$1::uuid AND o_que LIKE '%Sentença transitada%'`, id) != 1 {
		t.Error("o motivo do Mover estado continua indo ao andamento (Plano 2)")
	}
	if e.moverProcesso(adv, id, "ativo", "") != 200 {
		t.Fatal("reativar")
	}
	if ab, tot := e.monitor(id); ab != 1 || tot != 2 {
		t.Errorf("extinto→ativo cria NOVA tarefa; a antiga permanece: %d/%d", ab, tot)
	}
	if e.moverProcesso(adv, id, "arquivado", "") != 200 {
		t.Fatal("arquivar")
	}
	if ab, tot := e.monitor(id); ab != 0 || tot != 2 {
		t.Errorf("arquivado baixa também: %d/%d", ab, tot)
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='monitoramento.criado'`) != 2 || e.count(`SELECT count(*) FROM audit_events WHERE action='monitoramento.baixado'`) != 2 {
		t.Error("auditoria registra monitoramento.criado e monitoramento.baixado")
	}
	if e.count(`SELECT count(*) FROM tarefas WHERE monitoramento AND (prazo_fatal IS NOT NULL OR prazo IS NOT NULL)`) != 0 {
		t.Error("a tarefa de monitoramento não tem prazo")
	}
}

func TestMonitoramentoProcessoSemNumeroTemAviso(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv-sn@exemplo.test", auth.Advogado)
	id := adv.criarProcesso(t, "")
	var desc string
	_ = e.pool.QueryRow(context.Background(), `SELECT descricao FROM tarefas WHERE monitoramento AND processo_id=$1::uuid`, id).Scan(&desc)
	if !strings.Contains(desc, "ATENÇÃO: ficha sem número CNJ") {
		t.Errorf("descrição: %q", desc)
	}
}

type monitoramentoQueFalha struct{ processos.Monitoramento }

func (monitoramentoQueFalha) Baixar(context.Context, pgx.Tx, webx.Actor, string, string) (bool, error) {
	return false, errors.New("falha forçada")
}

func TestFalhaNaTarefaDesfazAMudancaDeEstado(t *testing.T) {
	e := newEnvOpts(t, WithMonitoramento(func(m processos.Monitoramento) processos.Monitoramento { return monitoramentoQueFalha{m} }))
	adv, _ := e.user("adv-fl@exemplo.test", auth.Advogado)
	id := adv.criarProcesso(t, numA)
	if e.moverProcesso(adv, id, "extinto", "") == 200 {
		t.Fatal("a falha na tarefa deveria recusar a mudança")
	}
	var sit string
	_ = e.pool.QueryRow(context.Background(), `SELECT situacao FROM processos WHERE id=$1::uuid`, id).Scan(&sit)
	if ab, _ := e.monitor(id); sit != "ativo" || ab != 1 || e.count(`SELECT count(*) FROM processo_andamentos WHERE processo_id=$1::uuid`, id) != 0 {
		t.Errorf("tudo ou nada: situacao=%s abertas=%d", sit, ab)
	}
}

// semMonitoramento tira do banco de teste as tarefas de monitoramento já criadas e devolve o contador de códigos ao ponto anterior,
// para os testes antigos (que contam tarefas e códigos) não dependerem da tarefa automática.
func (e *env) semMonitoramento() {
	e.t.Helper()
	ctx := context.Background()
	var n int
	_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM tarefas WHERE monitoramento`).Scan(&n)
	_, _ = e.pool.Exec(ctx, `DELETE FROM tarefas WHERE monitoramento`)
	_, _ = e.pool.Exec(ctx, `UPDATE codigos_contador SET ultimo = ultimo - $1 WHERE prefixo='TAR'`, n)
}
