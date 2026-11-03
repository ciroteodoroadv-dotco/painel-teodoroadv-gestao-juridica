package app

import (
	"context"
	"painelteodoro/internal/auth"
	"testing"
)

func TestMigracoes0007(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	_, uid := e.user("m7@exemplo.test", auth.Advogado)
	// Sem isto, as linhas com valores novos impedem o Down da migração 0007 no Reset do teste seguinte.
	t.Cleanup(func() {
		_, _ = e.pool.Exec(ctx, `DELETE FROM agenda_eventos`)
	})
	ev := func(tipo string) error {
		_, err := e.pool.Exec(ctx, `INSERT INTO agenda_eventos (tipo, titulo, inicio, created_by, updated_by) VALUES ($1,'Compromisso de teste', now(), $2, $2)`, tipo, uid)
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
}
