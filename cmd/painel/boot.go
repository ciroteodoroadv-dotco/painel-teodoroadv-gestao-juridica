package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/platform/config"
	"painelteodoro/internal/platform/db"
	"painelteodoro/internal/platform/migrate"
)

// openApp abre só o pool do papel restrito: é o caminho do servidor web quando ele não recebe a senha do dono
// (produção, auditoria 05/11 item 2.1). As migrações rodam antes, no serviço "migrate".
func openApp(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	app, err := db.Open(ctx, cfg.AppDatabaseURL)
	if err != nil {
		return nil, err
	}
	if err = app.Ping(ctx); err != nil {
		app.Close()
		return nil, fmt.Errorf("o papel restrito do app não conectou ao banco: rode antes \"painel migrate up\" (serviço migrate do Compose)")
	}
	return app, nil
}

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
