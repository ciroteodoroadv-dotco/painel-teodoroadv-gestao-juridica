// Package db abre o pool de conexões com o PostgreSQL.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
