package app

import (
	"context"
	"painelteodoro/internal/auth"
	"strings"
	"testing"
)

func TestMigracoes0007(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	_, uid := e.user("m7@exemplo.test", auth.Advogado)
	// Sem isto, as linhas com valores novos impedem o Down da migração 0007 no Reset do teste seguinte.
	t.Cleanup(func() {
		_, _ = e.pool.Exec(ctx, `DELETE FROM tarefas`)
		_, _ = e.pool.Exec(ctx, `DELETE FROM agenda_eventos`)
	})
	ev := func(tipo string) error {
		_, err := e.pool.Exec(ctx, `INSERT INTO agenda_eventos (tipo, titulo, inicio, created_by, updated_by) VALUES ($1,'Compromisso de teste', now(), $2, $2)`, tipo, uid)
		return err
	}
	tf := func(status, prioridade, subtitulo string) error {
		_, err := e.pool.Exec(ctx, `INSERT INTO tarefas (titulo, status, prioridade, subtitulo, created_by, updated_by) VALUES ('Tarefa de teste',$1,$2,$3,$4,$4)`,
			status, prioridade, subtitulo, uid)
		return err
	}
	for _, tipo := range []string{"aula", "ligacao"} {
		if err := ev(tipo); err != nil {
			t.Errorf("evento %q deveria ser aceito: %v", tipo, err)
		}
	}
	if err := ev("prazo_fatal"); err == nil {
		t.Error("tipo inexistente deveria ser recusado")
	}
	if err := tf("em_revisao", "alta", "Ligação"); err != nil {
		t.Errorf("tarefa em revisão/alta/subtítulo deveria ser aceita: %v", err)
	}
	if err := tf("aberta", "urgente", ""); err == nil {
		t.Error("prioridade 'urgente' deveria ser recusada")
	}
	if err := tf("aberta", "normal", strings.Repeat("a", 81)); err == nil {
		t.Error("subtítulo de 81 caracteres deveria ser recusado")
	}
	if err := tf("aberta", "normal", strings.Repeat("a", 80)); err != nil {
		t.Errorf("subtítulo de 80 caracteres deveria ser aceito: %v", err)
	}
}
