package migrate_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/platform/migrate"
)

// TestCicloUpDownUp prova o ciclo exigido no M1: aplicar tudo, reverter a última,
// reaplicar, reverter tudo e aplicar de novo. Roda no job "migracoes" do CI
// com PAINEL_TEST_DB apontando para um PostgreSQL descartável.
func TestCicloUpDownUp(t *testing.T) {
	dsn := os.Getenv("PAINEL_TEST_DB")
	if dsn == "" {
		t.Skip("PAINEL_TEST_DB não definida: teste de integração pulado")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	if err := migrate.Reset(ctx, pool); err != nil {
		t.Fatalf("reverter tudo: %v", err)
	}
	if err := migrate.Up(ctx, pool); err != nil {
		t.Fatalf("aplicar tudo: %v", err)
	}
	if err := migrate.Down(ctx, pool); err != nil {
		t.Fatalf("reverter a última: %v", err)
	}
	if err := migrate.Up(ctx, pool); err != nil {
		t.Fatalf("reaplicar a última: %v", err)
	}
	if err := migrate.Reset(ctx, pool); err != nil {
		t.Fatalf("reverter tudo pela segunda vez: %v", err)
	}
	if err := migrate.Up(ctx, pool); err != nil {
		t.Fatalf("aplicar tudo pela segunda vez: %v", err)
	}
	var existe bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.audit_events') IS NOT NULL`).Scan(&existe); err != nil {
		t.Fatal(err)
	}
	if !existe {
		t.Fatal("audit_events não existe depois do ciclo up→down→up")
	}
}
