package audit

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/platform/db"
)

// O app roda sem poder de dono: na auditoria só enxerga e acrescenta. Prova direta no banco.
func TestPapelDoAppNaoAlteraAuditoria(t *testing.T) {
	ctx := context.Background()
	owner := testPool(t)
	if err := db.SetAppPassword(ctx, owner, "senha-de-teste-do-papel"); err != nil {
		t.Fatal(err)
	}
	cfg := owner.Config()
	cc := cfg.ConnConfig.Copy()
	cc.User, cc.Password = db.AppRole, "senha-de-teste-do-papel"
	pc, _ := pgxpool.ParseConfig(cc.ConnString())
	pc.ConnConfig = cc
	app, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	if err := RecordPool(ctx, app, Event{Actor: "u1", Action: "teste", Result: OK}); err != nil {
		t.Fatalf("o papel do app deveria conseguir INSERT na auditoria: %v", err)
	}
	if n, err := Verify(ctx, app); err != nil || n != 1 {
		t.Fatalf("o papel do app deveria conseguir ler e verificar: n=%d err=%v", n, err)
	}
	proibido := []string{
		"UPDATE audit_events SET actor='x'",
		"DELETE FROM audit_events",
		"TRUNCATE audit_events",
		"ALTER TABLE audit_events DISABLE TRIGGER audit_events_sem_update_delete",
		"DROP TABLE audit_events",
		"CREATE TABLE intruso (x int)",
		"INSERT INTO goose_db_version (version_id, is_applied) VALUES (99, true)",
		"ALTER ROLE painel_app SUPERUSER",
	}
	for _, q := range proibido {
		_, err := app.Exec(ctx, q)
		if err == nil {
			t.Errorf("%q deveria ser negado ao papel do app", q)
		} else if !strings.Contains(err.Error(), "permission denied") && !strings.Contains(err.Error(), "must be owner") &&
			!strings.Contains(err.Error(), "bloqueado") {
			t.Errorf("%q falhou por motivo inesperado: %v", q, err)
		}
	}
	// Operação normal do app em outras tabelas continua permitida.
	if _, err := app.Exec(ctx, "INSERT INTO users (email, role, password_hash) VALUES ('p@x.test','advogado','h')"); err != nil {
		t.Errorf("o app deveria gravar em users: %v", err)
	}
}
