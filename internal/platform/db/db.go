// Package db abre o pool de conexões com o PostgreSQL.
package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AppRole é o papel de banco usado em tempo de execução (criado pela migração 0003).
const AppRole = "painel_app"

// Open cria o pool. Não conecta de imediato: a prontidão é checada por Ping em /readyz.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		// Não inclui o erro original: ele pode ecoar a URL com a senha.
		return nil, fmt.Errorf("configuração do banco inválida")
	}
	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	return pgxpool.NewWithConfig(ctx, cfg)
}

// SetAppPassword define a senha do papel do app. Deve ser chamado com o pool do DONO do banco,
// depois das migrações. O valor é escapado pelo próprio PostgreSQL (format %L).
func SetAppPassword(ctx context.Context, owner *pgxpool.Pool, password string) error {
	if password == "" {
		return errors.New("senha do papel do app vazia")
	}
	var stmt string
	err := owner.QueryRow(ctx, "SELECT format('ALTER ROLE %I WITH LOGIN PASSWORD %L', $1::text, $2::text)",
		AppRole, password).Scan(&stmt)
	if err != nil {
		return err
	}
	_, err = owner.Exec(ctx, stmt)
	return err
}
