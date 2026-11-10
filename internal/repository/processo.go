package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Processo struct {
	ID        int64     `json:"id"`
	Numero    string    `json:"numero"`
	Cliente   string    `json:"cliente"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type ProcessoRepository struct {
	db *pgxpool.Pool
}

func NewProcessoRepository(db *pgxpool.Pool) *ProcessoRepository {
	return &ProcessoRepository{db: db}
}

func (r *ProcessoRepository) Contar(ctx context.Context) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("pool de conexao nao inicializado")
	}

	var total int64
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM processos").Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("erro ao contar processos: %w", err)
	}

	return total, nil
}
