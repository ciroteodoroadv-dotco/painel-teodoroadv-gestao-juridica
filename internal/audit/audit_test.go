package audit

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/platform/migrate"
)

// testPool exige PAINEL_TEST_DB (um PostgreSQL descartável). Sem ela, o teste é pulado.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("PAINEL_TEST_DB")
	if url == "" {
		t.Skip("PAINEL_TEST_DB não definida: teste de integração pulado")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	// Recomeça do zero: reverte (se houver) e aplica de novo — também prova up→down→up.
	if err := migrate.Reset(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Up(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestCadeiaEImutabilidade(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	for i := 0; i < 3; i++ {
		ev := Event{Actor: "u1", Action: "teste", Result: OK, RequestID: "r",
			Meta: map[string]string{"motivo": "x", "cpf": "000.000.000-00"}}
		if err := RecordPool(ctx, pool, ev); err != nil {
			t.Fatal(err)
		}
	}
	n, err := Verify(ctx, pool)
	if err != nil || n != 3 {
		t.Fatalf("Verify = %d, %v", n, err)
	}
	var meta string
	if err := pool.QueryRow(ctx, "SELECT meta::text FROM audit_events LIMIT 1").Scan(&meta); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(meta, "cpf") || strings.Contains(meta, "000.000") {
		t.Fatalf("metadado fora da lista permitida foi gravado: %s", meta)
	}

	for _, q := range []string{
		"UPDATE audit_events SET actor='x'",
		"DELETE FROM audit_events",
		"TRUNCATE audit_events",
	} {
		if _, err := pool.Exec(ctx, q); err == nil {
			t.Errorf("%q deveria ser bloqueado pelo gatilho", q)
		}
	}
}

func TestVerifyDetectaAdulteracao(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	for i := 0; i < 2; i++ {
		_ = RecordPool(ctx, pool, Event{Actor: "u1", Action: "a", Result: OK})
	}
	// Quem tem poder de superusuário consegue desligar o gatilho; a cadeia deve denunciar.
	tx, _ := pool.Begin(ctx)
	_, _ = tx.Exec(ctx, "ALTER TABLE audit_events DISABLE TRIGGER audit_events_sem_update_delete")
	_, _ = tx.Exec(ctx, "UPDATE audit_events SET actor='intruso' WHERE seq=(SELECT min(seq) FROM audit_events)")
	_, _ = tx.Exec(ctx, "ALTER TABLE audit_events ENABLE TRIGGER audit_events_sem_update_delete")
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(ctx, pool); err == nil {
		t.Fatal("Verify deveria detectar a adulteração")
	}
}

func TestRecordExigeCampos(t *testing.T) {
	pool := testPool(t)
	if err := RecordPool(context.Background(), pool, Event{Action: "a", Result: OK}); err == nil {
		t.Fatal("evento sem ator deveria ser recusado")
	}
}
