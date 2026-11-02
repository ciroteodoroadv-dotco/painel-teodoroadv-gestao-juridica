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
	if e.count(`SELECT count(*) FROM pessoas`) != 4 || e.count(`SELECT count(*) FROM processos`) != 2 ||
		e.count(`SELECT count(*) FROM tarefas`) != 4 || e.count(`SELECT count(*) FROM agenda_eventos`) != 1 || e.count(`SELECT count(*) FROM processo_partes`) != 3 {
		t.Fatal("quantidades inesperadas na demonstração")
	}
	if err := SeedDemo(ctx, e.pool, fieldKey, e.a.Store, id); err == nil {
		t.Fatal("a demonstração não pode rodar duas vezes")
	}
	if _, body := adv.get("/"); !strings.Contains(body, "Exemplo: revisar contestação") {
		t.Error("a tela Hoje deveria mostrar a tarefa atrasada da demonstração")
	}
}
