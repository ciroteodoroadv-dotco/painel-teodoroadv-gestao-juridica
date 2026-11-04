package processos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

// MonitorRef é o processo como o monitor do DataJud o enxerga.
type MonitorRef struct{ ID, Numero, CriadoPor string }

// ListarParaMonitor devolve TODOS os processos não excluídos, em qualquer estado (E1 do Plano 6); quem decide se o número é
// consultável é o monitor.
func (s *Service) ListarParaMonitor(ctx context.Context) ([]MonitorRef, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id::text, numero_cnj, created_by::text FROM processos WHERE excluido_em IS NULL ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MonitorRef
	for rows.Next() {
		var m MonitorRef
		if err := rows.Scan(&m.ID, &m.Numero, &m.CriadoPor); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// RefParaMonitor devolve um processo não excluído.
func (s *Service) RefParaMonitor(ctx context.Context, id string) (MonitorRef, bool, error) {
	if !webx.UUID(id) {
		return MonitorRef{}, false, nil
	}
	var m MonitorRef
	err := s.Pool.QueryRow(ctx, `SELECT id::text, numero_cnj, created_by::text FROM processos WHERE id=$1::uuid AND excluido_em IS NULL`, id).Scan(&m.ID, &m.Numero, &m.CriadoPor)
	if errors.Is(err, pgx.ErrNoRows) {
		return MonitorRef{}, false, nil
	}
	return m, err == nil, err
}
