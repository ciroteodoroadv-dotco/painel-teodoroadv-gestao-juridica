package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/platform/config"
	"painelteodoro/internal/platform/db"
	"painelteodoro/internal/platform/migrate"
)

// openDBs prepara o banco: aplica as migrações com a conexão do DONO, define a senha do papel
// restrito e devolve também o pool desse papel — o único que o app usa em tempo de execução.
func openDBs(ctx context.Context, cfg config.Config) (owner, app *pgxpool.Pool, err error) {
	owner, err = db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	if err = migrate.Up(ctx, owner); err != nil {
		owner.Close()
		return nil, nil, fmt.Errorf("migração do banco falhou: %w", err)
	}
	if err = db.SetAppPassword(ctx, owner, cfg.AppDBPassword); err != nil {
		owner.Close()
		return nil, nil, fmt.Errorf("não consegui preparar o papel do app: %w", err)
	}
	if app, err = db.Open(ctx, cfg.AppDatabaseURL); err != nil {
		owner.Close()
		return nil, nil, err
	}
	if err = app.Ping(ctx); err != nil {
		owner.Close()
		app.Close()
		return nil, nil, fmt.Errorf("o papel restrito do app não conectou ao banco")
	}
	return owner, app, nil
}
