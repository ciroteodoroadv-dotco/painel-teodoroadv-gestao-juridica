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

// Down reverte a última migração.
func Down(ctx context.Context, pool *pgxpool.Pool) error {
	db, err := open(pool)
	if err != nil {
		return err
	}
	defer db.Close()
	return goose.DownContext(ctx, db, ".")
}
