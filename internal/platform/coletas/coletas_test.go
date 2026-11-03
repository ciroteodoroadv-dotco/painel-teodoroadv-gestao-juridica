package coletas_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/platform/coletas"
	"painelteodoro/internal/platform/migrate"
)

func store(t *testing.T) *coletas.Store {
	t.Helper()
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
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `TRUNCATE intimacoes, oab_monitoradas, coletas CASCADE`)
	})
	_, _ = pool.Exec(ctx, `TRUNCATE pessoas, processos, atendimentos, agenda_compromissos, audiencias, diligencias, intimacoes, oab_monitoradas, coletas CASCADE`)
	if err := migrate.Reset(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Up(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return &coletas.Store{Pool: pool}
}

func ptr(t time.Time) *time.Time { return &t }

func TestRegistrarSoAvancaSucessoEmSucesso(t *testing.T) {
	s := store(t)
	ctx := context.Background()
	t0 := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	d0 := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	if _, ok, err := s.Obter(ctx, "djen"); err != nil || ok {
		t.Fatalf("antes de registrar: ok=%v err=%v", ok, err)
	}
	if err := s.Registrar(ctx, coletas.Estado{Modulo: "djen", Origem: "DJEN", Status: "sucesso", Mensagem: "2 novas", UltimaTentativa: ptr(t0), JanelaDesde: ptr(d0)}); err != nil {
		t.Fatal(err)
	}
	e, _, _ := s.Obter(ctx, "djen")
	if e.UltimoSucesso == nil || !e.UltimoSucesso.Equal(t0) || e.JanelaDesde == nil {
		t.Fatalf("sucesso deveria avançar último sucesso e janela: %+v", e)
	}
	t1 := t0.Add(time.Hour)
	for _, st := range []string{"falha", "parcial", "bloqueado"} {
		if err := s.Registrar(ctx, coletas.Estado{Modulo: "djen", Origem: "DJEN", Status: st, Mensagem: "x", UltimaTentativa: ptr(t1), JanelaDesde: ptr(d0.AddDate(0, 0, 5))}); err != nil {
			t.Fatal(err)
		}
		e, _, _ = s.Obter(ctx, "djen")
		if e.Status != st || !e.UltimaTentativa.Equal(t1) {
			t.Errorf("%s: tentativa e status deveriam ser atualizados: %+v", st, e)
		}
		if !e.UltimoSucesso.Equal(t0) {
			t.Errorf("%s: o último sucesso não pode avançar: %v", st, e.UltimoSucesso)
		}
		if !e.JanelaDesde.Equal(d0) {
			t.Errorf("%s: a janela não pode avançar: %v", st, e.JanelaDesde)
		}
	}
	t2 := t1.Add(time.Hour)
	_ = s.Registrar(ctx, coletas.Estado{Modulo: "djen", Origem: "DJEN", Status: "sucesso", UltimaTentativa: ptr(t2), JanelaDesde: ptr(d0.AddDate(0, 0, 6))})
	e, _, _ = s.Obter(ctx, "djen")
	if !e.UltimoSucesso.Equal(t2) || !e.JanelaDesde.Equal(d0.AddDate(0, 0, 6)) {
		t.Errorf("novo sucesso deveria avançar: %+v", e)
	}
	if l, err := s.Listar(ctx); err != nil || len(l) != 1 {
		t.Errorf("Listar = %d %v", len(l), err)
	}
	if err := s.Registrar(ctx, coletas.Estado{Modulo: "x", Status: "ok"}); err == nil {
		t.Error("status inválido deveria ser recusado")
	}
}
