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

func (r *ProcessoRepository) Listar(ctx context.Context, limit, offset int) ([]Processo, error) {
	if r.db == nil {
		return nil, fmt.Errorf("pool de conexao nao inicializado")
	}

	rows, err := r.db.Query(ctx, "SELECT id, numero, cliente, status, created_at FROM processos ORDER BY id DESC LIMIT $1 OFFSET $2", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar processos: %w", err)
	}
	defer rows.Close()

	var lista []Processo
	for rows.Next() {
		var p Processo
		if err := rows.Scan(&p.ID, &p.Numero, &p.Cliente, &p.Status, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("erro ao ler linha de processo: %w", err)
		}
		lista = append(lista, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar processos: %w", err)
	}

	return lista, nil
}
