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
		e.count(`SELECT count(*) FROM tarefas`) != 6 || e.count(`SELECT count(*) FROM atendimentos`) != 3 ||
		e.count(`SELECT count(*) FROM tarefas WHERE tarefa_pai_id IS NOT NULL`) != 1 || e.count(`SELECT count(*) FROM prazos WHERE tarefa_id IS NOT NULL AND situacao='a_conferir'`) != 1 || e.count(`SELECT count(*) FROM agenda_compromissos`) != 3 || e.count(`SELECT count(*) FROM audiencias`) != 2 || e.count(`SELECT count(*) FROM audiencias WHERE cancelada`) != 1 || e.count(`SELECT count(*) FROM diligencias`) != 2 || e.count(`SELECT count(*) FROM processo_partes`) != 6 {
		t.Fatal("quantidades inesperadas na demonstração")
	}
	if err := SeedDemo(ctx, e.pool, fieldKey, e.a.Store, id); err == nil {
		t.Fatal("a demonstração não pode rodar duas vezes")
	}
	if _, body := adv.get("/tarefas"); !strings.Contains(body, "Protocolar embargos à execução (exemplo)") || !strings.Contains(body, "A CONFERIR") {
		t.Error("a tela de Tarefas deveria mostrar a demonstração, com o selo A CONFERIR")
	}
}
