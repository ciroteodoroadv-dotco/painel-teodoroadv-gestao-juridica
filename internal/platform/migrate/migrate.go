// Package migrate aplica e reverte migrações SQL (goose) embutidas no binário.
package migrate

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"painelteodoro/migrations"
)

func open(pool *pgxpool.Pool) (*sql.DB, error) {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return nil, err
	}
	return stdlib.OpenDBFromPool(pool), nil
}

// Up aplica todas as migrações pendentes.
func Up(ctx context.Context, pool *pgxpool.Pool) error {
	db, err := open(pool)
	if err != nil {
		return err
	}
	defer db.Close()
	return goose.UpContext(ctx, db, ".")
}

// Reset reverte TODAS as migrações (uso exclusivo de testes: apaga o esquema inteiro).
func Reset(ctx context.Context, pool *pgxpool.Pool) error {
	db, err := open(pool)
	if err != nil {
		return err
	}
	defer db.Close()
	// O Down da 0007 recusa de propósito linhas com valores novos (não perde dado em silêncio).
	// Reset é exclusivo de testes, então esvazia essas duas tabelas antes de reverter.
	_, _ = db.ExecContext(ctx, `DO $$ BEGIN
		IF to_regclass('public.tarefas') IS NOT NULL THEN DELETE FROM tarefas; END IF;
		IF to_regclass('public.agenda_eventos') IS NOT NULL THEN DELETE FROM agenda_eventos; END IF;
	END $$`)
	return goose.DownToContext(ctx, db, ".", 0)
}

// Down reverte a última migração.
func Down(ctx context.Context, pool *pgxpool.Pool) error {
	db, err := open(pool)
	if err != nil {
		return err
	}
	defer db.Close()
	return goose.DownContext(ctx, db, ".")
}
