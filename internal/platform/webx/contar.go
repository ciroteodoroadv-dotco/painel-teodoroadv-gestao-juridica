package webx

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ContarAtivos devolve a função que conta as linhas não excluídas de uma tabela (contador do menu).
// tabela é sempre uma constante do código, nunca texto vindo do usuário.
func ContarAtivos(pool *pgxpool.Pool, tabela string) func(context.Context) (int, error) {
	return func(ctx context.Context) (int, error) {
		var n int
		err := pool.QueryRow(ctx, "SELECT count(*) FROM "+tabela+" WHERE excluido_em IS NULL").Scan(&n) // #nosec G202 -- tabela constante
		return n, err
	}
}
