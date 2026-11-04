package app

import (
	"context"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
)

func TestSeedDemo(t *testing.T) {
	e := newEnv(t)
	adv, id := e.user("adv@exemplo.test", auth.Advogado)
	ctx := context.Background()
	if err := SeedDemo(ctx, e.pool, fieldKey, e.a.Store, id); err != nil {
		t.Fatal(err)
	}
	if e.count(`SELECT count(*) FROM pessoas`) != 4 || e.count(`SELECT count(*) FROM processos`) != 3 ||
		e.count(`SELECT count(*) FROM tarefas`) != 12 || e.count(`SELECT count(*) FROM atendimentos`) != 3 ||
		e.count(`SELECT count(*) FROM tarefas WHERE tarefa_pai_id IS NOT NULL`) != 1 || e.count(`SELECT count(*) FROM prazos WHERE tarefa_id IS NOT NULL AND situacao='a_conferir'`) != 5 || e.count(`SELECT count(*) FROM agenda_compromissos`) != 3 || e.count(`SELECT count(*) FROM audiencias`) != 2 || e.count(`SELECT count(*) FROM audiencias WHERE cancelada`) != 1 || e.count(`SELECT count(*) FROM diligencias`) != 2 || e.count(`SELECT count(*) FROM processo_partes`) != 6 {
		t.Fatal("quantidades inesperadas na demonstração")
	}
	if e.count(`SELECT count(*) FROM intimacoes`) != 5 || e.count(`SELECT count(*) FROM intimacoes WHERE status='triagem'`) != 3 ||
		e.count(`SELECT count(*) FROM intimacoes WHERE status='confirmada'`) != 1 || e.count(`SELECT count(*) FROM intimacoes WHERE status='cancelada'`) != 1 ||
		e.count(`SELECT count(*) FROM tarefas WHERE intimacao_id IS NOT NULL AND estado='a-confirmar' AND prazo_fatal IS NOT NULL AND prazo_interno`) != 3 ||
		e.count(`SELECT count(*) FROM oab_monitoradas`) != 1 || e.count(`SELECT count(*) FROM coletas`) != 3 ||
		e.count(`SELECT count(*) FROM coletas WHERE status='sucesso'`) != 1 || e.count(`SELECT count(*) FROM coletas WHERE status='parcial'`) != 1 {
		t.Fatal("a demonstração de intimações deveria ter 1 OAB, 5 intimações (3 do DJEN, uma por status, e 2 do DataJud em triagem), 5 tarefas de análise e 3 coletas")
	}
	// Plano 7: calendários fictícios. A Justiça Federal vem [VERIFICAR] e SEM efeito no cálculo; o de teste tem 2 dias com efeito.
	if e.count(`SELECT count(*) FROM calendarios WHERE sigla IN ('TRF-DEMO','TRIBUNAL DE TESTE')`) != 2 ||
		e.count(`SELECT count(*) FROM calendario_dias d JOIN calendarios c ON c.id=d.calendario_id WHERE c.sigla='TRIBUNAL DE TESTE' AND d.conta_como_nao_util`) != 2 ||
		e.count(`SELECT count(*) FROM calendario_dias d JOIN calendarios c ON c.id=d.calendario_id WHERE c.sigla='TRF-DEMO' AND d.conta_como_nao_util`) != 0 ||
		e.count(`SELECT count(*) FROM calendarios WHERE sigla='TRF-DEMO' AND verificar`) != 1 {
		t.Fatal("a demonstração deveria trazer o calendário da Justiça Federal [VERIFICAR] (sem efeito) e o TRIBUNAL DE TESTE (2 dias)")
	}
	if err := SeedDemo(ctx, e.pool, fieldKey, e.a.Store, id); err == nil {
		t.Fatal("a demonstração não pode rodar duas vezes")
	}
	if _, body := adv.get("/tarefas"); !strings.Contains(body, "Protocolar embargos à execução (exemplo)") || !strings.Contains(body, "A CONFERIR") {
		t.Error("a tela de Tarefas deveria mostrar a demonstração, com o selo A CONFERIR")
	}
}
