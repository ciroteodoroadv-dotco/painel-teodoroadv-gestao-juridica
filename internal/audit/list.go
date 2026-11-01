package audit

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Row é uma linha para exibição (sem metadados: a tela mostra só o essencial).
type Row struct {
	Seq        int64
	At         time.Time
	Actor      string
	Action     string
	Result     string
	RequestID  string
	TargetType string
}

// Recent devolve os n eventos mais recentes.
func Recent(ctx context.Context, pool *pgxpool.Pool, n int) ([]Row, error) {
	rows, err := pool.Query(ctx, `SELECT seq, at, actor, action, result, request_id, target_type
		FROM audit_events ORDER BY seq DESC LIMIT $1`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.Seq, &r.At, &r.Actor, &r.Action, &r.Result, &r.RequestID, &r.TargetType); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
