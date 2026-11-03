package codigos_test

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/platform/codigos"
	"painelteodoro/internal/platform/migrate"
)

func pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("PAINEL_TEST_DB")
	if dsn == "" {
		t.Skip("PAINEL_TEST_DB não definida: teste de integração pulado")
	}
	ctx := context.Background()
	p, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	_, _ = p.Exec(ctx, `TRUNCATE pessoas, processos CASCADE`) // a migração 0008 só reverte com tabelas vazias
	if err := migrate.Reset(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Up(ctx, p); err != nil {
		t.Fatal(err)
	}
	return p
}

func proximo(t *testing.T, p *pgxpool.Pool, prefixo string) string {
	t.Helper()
	ctx := context.Background()
	tx, err := p.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	c, err := codigos.Proximo(ctx, tx, prefixo)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSequenciaPorPrefixo(t *testing.T) {
	p := pool(t)
	for i, esperado := range []string{"PES. 0000001", "PES. 0000002"} {
		if got := proximo(t, p, "PES"); got != esperado {
			t.Errorf("chamada %d: %q, esperado %q", i, got, esperado)
		}
	}
	if got := proximo(t, p, "PRO"); got != "PRO. 0000001" {
		t.Errorf("contador de PRO deve ser independente, veio %q", got)
	}
}

func TestPrefixoInvalido(t *testing.T) {
	p := pool(t)
	tx, _ := p.Begin(context.Background())
	defer func() { _ = tx.Rollback(context.Background()) }()
	for _, ruim := range []string{"", "pes", "PE", "PESS", "PE1", "PES'; DROP TABLE pessoas;--"} {
		if _, err := codigos.Proximo(context.Background(), tx, ruim); err == nil {
			t.Errorf("prefixo %q deveria ser recusado", ruim)
		}
	}
}

func TestRollbackNaoConsomeNumero(t *testing.T) {
	p := pool(t)
	ctx := context.Background()
	tx, _ := p.Begin(ctx)
	if c, err := codigos.Proximo(ctx, tx, "ATE"); err != nil || c != "ATE. 0000001" {
		t.Fatalf("1ª chamada: %q %v", c, err)
	}
	_ = tx.Rollback(ctx)
	if got := proximo(t, p, "ATE"); got != "ATE. 0000001" {
		t.Errorf("após rollback o número deve ser reaproveitado, veio %q", got)
	}
}

func TestConcorrenciaNaoRepete(t *testing.T) {
	p := pool(t)
	const n = 20
	var wg sync.WaitGroup
	var mu sync.Mutex
	vistos := map[string]int{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := proximo(t, p, "TAR")
			mu.Lock()
			vistos[c]++
			mu.Unlock()
		}()
	}
	wg.Wait()
	if len(vistos) != n {
		t.Fatalf("esperava %d códigos distintos, veio %d: %v", n, len(vistos), vistos)
	}
}
